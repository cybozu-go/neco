package main

import (
	"errors"
	"fmt"
	"net"
)

// Cluster is the network configurations of a Kubernetes cluster.
type Cluster struct {
	Name       string   `json:"name"`
	Bastion    string   `json:"bastion_network"`
	BMC        string   `json:"bmc_network"`
	NTPServers []string `json:"ntp_servers"`
}

// Validate validates the configurations
func (c Cluster) Validate() error {
	if len(c.Name) == 0 {
		return errors.New("no name")
	}

	if ip := net.ParseIP(c.Bastion); ip == nil {
		return fmt.Errorf("invalid bastion network: %s", c.Bastion)
	}

	if _, _, err := net.ParseCIDR(c.BMC); err != nil {
		return fmt.Errorf("invalid BMC network: %w", err)
	}

	if len(c.NTPServers) == 0 {
		return errors.New("no NTP servers")
	}

	for _, n := range c.NTPServers {
		if ip := net.ParseIP(n); ip == nil {
			return fmt.Errorf("invalid NTP server address: %s", n)
		}
	}

	return nil
}
