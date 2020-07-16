package placemat

import (
	"context"
	"errors"
	"net"
)

const (
	maxNetworkNameLen = 15
	v4ForwardKey      = "net.ipv4.ip_forward"
	v6ForwardKey      = "net.ipv6.conf.all.forwarding"
)

// NetworkType represents a network type.
type NetworkType int

// Network types.
const (
	NetworkInternal NetworkType = iota
	NetworkExternal
	NetworkBMC
)

// NetworkSpec represents a Network specification in YAML
type NetworkSpec struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	UseNAT  bool   `json:"use-nat"`
	Address string `json:"address,omitempty"`
}

// Network represents a network configuration
type Network struct {
	*NetworkSpec

	typ         NetworkType
	ip          net.IP
	ipNet       *net.IPNet
	tapNames    []string
	vethNames   []string
	ng          *nameGenerator
	v4forwarded bool
	v6forwarded bool
}

// NewNetwork creates *Network from spec.
func NewNetwork(spec *NetworkSpec) (*Network, error) {
	n := &Network{
		NetworkSpec: spec,
	}

	if len(spec.Name) > maxNetworkNameLen {
		return nil, errors.New("too long name: " + spec.Name)
	}

	switch spec.Type {
	case "internal":
		n.typ = NetworkInternal
		if spec.UseNAT {
			return nil, errors.New("UseNAT must be false for internal network")
		}
		if len(spec.Address) > 0 {
			return nil, errors.New("Address cannot be specified for internal network")
		}
	case "external":
		n.typ = NetworkExternal
		if len(spec.Address) == 0 {
			return nil, errors.New("Address must be specified for external network")
		}
	case "bmc":
		n.typ = NetworkBMC
		if spec.UseNAT {
			return nil, errors.New("UseNAT must be false for BMC network")
		}
		if len(spec.Address) == 0 {
			return nil, errors.New("Address must be specified for BMC network")
		}
	default:
		return nil, errors.New("unknown type: " + spec.Type)
	}

	if len(spec.Address) > 0 {
		ip, ipNet, err := net.ParseCIDR(spec.Address)
		if err != nil {
			return nil, err
		}
		n.ip = ip
		n.ipNet = ipNet
	}

	return n, nil
}

func iptables(ip net.IP) string {
	if ip.To4() != nil {
		return "iptables"
	}
	return "ip6tables"
}

func isForwarding(name string) bool {
	val, err := sysctlGet(name)
	if err != nil {
		return false
	}

	return len(val) > 0 && val[0] != '0'
}

func setForwarding(name string, flag bool) error {
	val := "1\n"
	if !flag {
		val = "0\n"
	}
	return sysctlSet(name, val)
}

// Create creates a virtual L2 switch using Linux bridge.
func (n *Network) Create(ng *nameGenerator) error {
	n.ng = ng

	cmds := [][]string{
		{"ip", "link", "add", n.Name, "type", "bridge"},
		{"ip", "link", "set", n.Name, "up"},
	}
	if len(n.Address) > 0 {
		cmds = append(cmds,
			[]string{"ip", "addr", "add", n.Address, "dev", n.Name},
		)
	}

	err := execCommands(context.Background(), cmds)
	if err != nil {
		return err
	}

	cmds = [][]string{
		{"iptables", "-t", "filter", "-A", "PLACEMAT", "-i", n.Name, "-j", "ACCEPT"},
		{"iptables", "-t", "filter", "-A", "PLACEMAT", "-o", n.Name, "-j", "ACCEPT"},
		{"ip6tables", "-t", "filter", "-A", "PLACEMAT", "-i", n.Name, "-j", "ACCEPT"},
		{"ip6tables", "-t", "filter", "-A", "PLACEMAT", "-o", n.Name, "-j", "ACCEPT"},
	}

	if !n.UseNAT {
		if n.Type == "internal" {
			return execCommands(context.Background(), cmds)
		}
		return nil
	}

	if !isForwarding(v4ForwardKey) {
		err = setForwarding(v4ForwardKey, true)
		if err != nil {
			return err
		}
		n.v4forwarded = true
	}

	if !isForwarding(v6ForwardKey) {
		err = setForwarding(v6ForwardKey, true)
		if err != nil {
			return err
		}
		n.v6forwarded = true
	}

	cmds = append(cmds, []string{iptables(n.ip), "-t", "nat", "-A", "PLACEMAT", "-j", "MASQUERADE",
		"--source", n.ipNet.String(), "!", "--destination", n.ipNet.String()})

	return execCommands(context.Background(), cmds)
}

// CreateTap add a tap device to the bridge and return the tap device name.
func (n *Network) CreateTap() (string, error) {
	name := n.ng.New()

	cmds := [][]string{
		{"ip", "tuntap", "add", name, "mode", "tap"},
		{"ip", "link", "set", name, "master", n.Name},
		{"ip", "link", "set", name, "up"},
	}
	err := execCommands(context.Background(), cmds)
	if err != nil {
		return "", err
	}

	n.tapNames = append(n.tapNames, name)
	return name, nil
}

// CreateVeth creates a veth pair and add one of the pair to the bridge.
// It returns both names of the pair.
func (n *Network) CreateVeth() (string, string, error) {
	name := n.ng.New()
	nameInNS := name + "_"

	cmds := [][]string{
		{"ip", "link", "add", name, "type", "veth", "peer", "name", nameInNS},
		{"ip", "link", "set", name, "master", n.Name, "up"},
	}
	err := execCommands(context.Background(), cmds)
	if err != nil {
		return "", "", err
	}

	n.vethNames = append(n.vethNames, name)
	return name, nameInNS, nil
}

// CleanupNetworks removes all remaining network resources.
func CleanupNetworks(r *Runtime, c *Cluster) {
	destroyNatRules()

	ng := &nameGenerator{
		prefix: r.ng.prefix,
	}
	var cmds [][]string

	for _, d := range c.Nodes {
		for range d.networks {
			name := ng.New()
			cmds = append(cmds, []string{"ip", "tuntap", "delete", name, "mode", "tap"})
		}
	}

	for _, p := range c.Pods {
		deletePodNS(context.Background(), p.Name)
		for range p.networks {
			name := ng.New()
			cmds = append(cmds, []string{"ip", "link", "delete", name})
		}
	}

	for _, n := range c.Networks {
		cmds = append(cmds, []string{"ip", "link", "delete", n.Name, "type", "bridge"})
	}

	// ignore all errors, because this commands will fail if tne network resource does not exist.
	execCommandsForce(cmds)
}

// Destroy deletes all created tap and veth devices, then the bridge.
func (n *Network) Destroy() error {
	if n.v4forwarded {
		setForwarding(v4ForwardKey, false)
	}
	if n.v6forwarded {
		setForwarding(v6ForwardKey, false)
	}

	cmds := [][]string{}
	for _, name := range n.tapNames {
		cmds = append(cmds, []string{"ip", "tuntap", "delete", name, "mode", "tap"})
	}
	for _, name := range n.vethNames {
		cmds = append(cmds, []string{"ip", "link", "delete", name})
	}
	cmds = append(cmds, []string{"ip", "link", "delete", n.Name, "type", "bridge"})

	return execCommandsForce(cmds)
}
