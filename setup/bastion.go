package setup

import (
	"errors"
	"net"
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
