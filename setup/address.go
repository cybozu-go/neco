package setup

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
)

func bastionIP() (net.IP, error) {
	iface, err := net.InterfaceByName("bastion")
	if err != nil {
		return nil, err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipnet.IP.To4() != nil {
			return ipnet.IP, nil
		}
	}

	return nil, errors.New("no IPv4 address for bastion device")
}

var bootIPList = []string{
	"10.71.0.0",
	"10.71.0.1",
	"10.71.0.2",
	"10.71.0.3",
	"10.71.0.4",
	"10.71.0.5",
	"10.71.0.6",
}

func setupBootIP(ctx context.Context, mylrn int, lrns []int) error {
	var addr string

	sort.Ints(lrns)
	for i, n := range lrns {
		if n == mylrn {
			addr = bootIPList[i]
			break
		}
	}

	s := fmt.Sprintf(`[Match]
Name=boot

[Network]
Address=%s/32
`, addr)
	err := os.WriteFile("/etc/systemd/network/boot.network", []byte(s), 0644)
	if err != nil {
		return fmt.Errorf("failed to create network file for boot: %w", err)
	}
	return nil
}
