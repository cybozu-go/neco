package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/netutil"
	"github.com/vishvananda/netlink"
)

const (
	baseASN      = 64600
	nodeMaskBits = 26
)

func setupNetwork(lrn int) error {
	fmt.Fprintln(os.Stderr, "Configuring networking...")
	node0, node1, node2, bastion := nodeAddresses(lrn)
	tor1 := torIP(node1)
	tor2 := torIP(node2)
	eth0, eth1, err := detectPhysLinks()
	if err != nil {
		return err
	}

	if err := writeDummyNetworkConf("node0", node0); err != nil {
		return err
	}
	if err := writeDummyNetworkConf("bastion", bastion); err != nil {
		return err
	}
	if err := writeDummyNetworkConf("boot"); err != nil {
		return err
	}
	if err := writePhysNetworkConf(eth0, node1); err != nil {
		return err
	}
	if err := writePhysNetworkConf(eth1, node2); err != nil {
		return err
	}

	if err := enableNetworkd(); err != nil {
		return err
	}

	if err := waitNetwork("node0", node0); err != nil {
		return err
	}
	if err := waitNetwork("bastion", bastion); err != nil {
		return err
	}
	if err := waitNetwork("boot"); err != nil {
		return err
	}
	if err := waitNetwork(eth0, node1); err != nil {
		return err
	}
	if err := waitNetwork(eth1, node2); err != nil {
		return err
	}

	if err := disableOffload(eth0, eth1); err != nil {
		return err
	}
	if err := configureDefaultRoute(tor1, tor2, bastion); err != nil {
		return err
	}
	if err := configureBird(baseASN+lrn, tor1, tor2); err != nil {
		return err
	}
	if err := configureChrony(); err != nil {
		return err
	}
	return nil
}

func writeDummyNetworkConf(name string, addrs ...net.IP) error {
	s := fmt.Sprintf(`[NetDev]
Name=%s
Kind=dummy
`, name)
	err := os.WriteFile(fmt.Sprintf("/etc/systemd/network/%s.netdev", name), []byte(s), 0644)
	if err != nil {
		return fmt.Errorf("failed to create netdev file for %s: %w", name, err)
	}

	if len(addrs) == 0 {
		return nil
	}

	builder := strings.Builder{}
	head := fmt.Sprintf(`[Match]
Name=%s

[Network]
`, name)
	builder.WriteString(head)
	for _, a := range addrs {
		builder.WriteString(fmt.Sprintf("Address=%s/32\n", a.String()))
	}
	err = os.WriteFile(fmt.Sprintf("/etc/systemd/network/%s.network", name), []byte(builder.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to create network file for %s: %w", name, err)
	}

	return nil
}

func writePhysNetworkConf(name string, addr net.IP) error {
	s := fmt.Sprintf(`[Match]
Name=%s

[Network]
LLDP=true
EmitLLDP=nearest-bridge

[Address]
Address=%s/26
Scope=link
`, name, addr)
	err := os.WriteFile(fmt.Sprintf("/etc/systemd/network/%s.network", name), []byte(s), 0644)
	if err != nil {
		return fmt.Errorf("failed to create network file for %s: %w", name, err)
	}
	return nil
}

func waitNetwork(linkName string, addrs ...net.IP) error {
	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return fmt.Errorf("failed to lookup %s: %w", linkName, err)
	}

	fmt.Fprintf(os.Stderr, "Waiting for the network interface %s to be ready...\n", linkName)

	expected := make([]string, 0, len(addrs))
	for _, a := range addrs {
		expected = append(expected, a.String())
	}
	slices.Sort(expected)

RETRY:
	al, _ := netlink.AddrList(link, netlink.FAMILY_V4)
	actual := make([]string, 0, len(al))
	for _, a := range al {
		actual = append(actual, a.IP.String())
	}
	slices.Sort(actual)

	if slices.Compare(expected, actual) != 0 {
		time.Sleep(1 * time.Second)
		goto RETRY
	}

	return nil
}

func nodeAddresses(lrn int) (node0, node1, node2, bastion net.IP) {
	node0 = neco.BootNode0IP(lrn)
	node1 = netutil.IPAdd(node0, 64)
	node2 = netutil.IPAdd(node0, 128)

	bastionBase := net.ParseIP(config.cluster.Bastion)
	bastion = netutil.IPAdd(bastionBase, int64(lrn))
	return
}

func detectPhysLinks() (eth0, eth1 string, err error) {
	err = exec.Command("systemd-detect-virt", "-v", "-q").Run()
	if err == nil {
		return "ens3", "ens4", nil
	}

	links, err := netlink.LinkList()
	if err != nil {
		return "", "", err
	}

	physLinks := make([]string, 0, len(links))
	for _, l := range links {
		dev, ok := l.(*netlink.Device)
		if !ok {
			continue
		}

		if dev.Name == "lo" || dev.Name == "idrac" {
			continue
		}

		if netlink.LinkSetUp(dev); err != nil {
			return "", "", fmt.Errorf("failed to set up %s: %w", dev.Name, err)
		}

		physLinks = append(physLinks, dev.Name)
	}

	fmt.Fprintf(os.Stderr, "  detected physical links: %v\n", physLinks)
	time.Sleep(10 * time.Second)

	upLinks := make([]string, 0, 2)
	for _, lname := range physLinks {
		l, err := netlink.LinkByName(lname)
		if err != nil {
			return "", "", fmt.Errorf("netlink: link not found %s: %w", lname, err)
		}

		if l.Attrs().OperState != netlink.OperUp {
			continue
		}
		upLinks = append(upLinks, lname)
	}

	if len(upLinks) != 2 {
		return "", "", errors.New("too few up links")
	}
	sort.Strings(upLinks)
	fmt.Fprintf(os.Stderr, "  configuring up links: %v\n", upLinks)
	return upLinks[0], upLinks[1], nil
}

func enableNetworkd() error {
	if err := os.RemoveAll("/etc/netplan"); err != nil {
		return fmt.Errorf("failed to remove /etc/netplan: %w", err)
	}

	if err := systemctl("enable", "systemd-networkd.service"); err != nil {
		return fmt.Errorf("failed to enable systemd-networkd: %w", err)
	}
	if err := systemctl("restart", "systemd-networkd.service"); err != nil {
		return fmt.Errorf("failed to restart systemd-networkd: %w", err)
	}

	return nil
}

func disableOffload(eth0, eth1 string) error {
	s := fmt.Sprintf(`[Unit]
Description=Disable network device offload
Wants=network-online.target
After=network-online.target
ConditionVirtualization=!kvm

[Service]
Type=oneshot
ExecStart=/sbin/ethtool -K %s tx off rx off
ExecStart=/sbin/ethtool -K %s tx off rx off
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
`, eth0, eth1)
	return installService("disable-offload", s)
}

func torIP(nodeIP net.IP) net.IP {
	base := nodeIP.Mask(net.CIDRMask(nodeMaskBits, 32))
	return netutil.IPAdd(base, 1)
}

func configureDefaultRoute(tor1, tor2, bastion net.IP) error {
	s := fmt.Sprintf(`[Unit]
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
ExecStart=/bin/ip route add 0.0.0.0/0 src %s nexthop via %s nexthop via %s
RemainAfterExit=yes
FailureAction=reboot-immediate

[Install]
WantedBy=multi-user.target
`, bastion, tor1, tor2)
	return installService("setup-route", s)
}

func configureBird(asn int, tor1, tor2 net.IP) error {
	buf := new(bytes.Buffer)
	err := birdConfTemplate.Execute(buf, struct {
		ASN  int
		Mask int
		ToR1 net.IP
		ToR2 net.IP
	}{
		asn,
		nodeMaskBits,
		tor1,
		tor2,
	})
	if err != nil {
		return err
	}

	err = os.WriteFile("/etc/bird/bird.conf", buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write /etc/bird/bird.conf: %w", err)
	}

	s := `[Unit]
Wants=network-online.target
After=network-online.target

[Service]
CPUSchedulingPolicy=rr
CPUSchedulingPriority=50
OOMScoreAdjust=-1000
`
	if err := installOverrideConf("bird", s); err != nil {
		return err
	}

	return enableService("bird")
}

func configureChrony() error {
	buf := new(bytes.Buffer)
	err := chronyConfTemplate.Execute(buf, config.cluster.NTPServers)
	if err != nil {
		return err
	}

	err = os.WriteFile("/etc/chrony/chrony.conf", buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write /etc/chrony/chrony.conf: %w", err)
	}

	s := `
[Unit]
Wants=network-online.target
After=network-online.target

[Service]
OOMScoreAdjust=-1000
LimitMEMLOCK=infinity
`
	if err := installOverrideConf("chrony", s); err != nil {
		return err
	}

	return enableService("chrony")
}
