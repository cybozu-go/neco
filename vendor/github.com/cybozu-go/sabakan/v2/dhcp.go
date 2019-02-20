package sabakan

import (
	"errors"
	"net"
	"time"
)

// DefaultLeaseDuration is 60 minutes.
const DefaultLeaseDuration = 60 * time.Minute

// DHCPConfig is a set of DHCP configurations.
type DHCPConfig struct {
	LeaseMinutes uint     `json:"lease-minutes"`
	DNSServers   []string `json:"dns-servers,omitempty"`

	// obsoleted fields
	GatewayOffset uint `json:"gateway-offset"`
}

// LeaseDuration returns lease duration for IP addreses.
func (c *DHCPConfig) LeaseDuration() time.Duration {
	if c.LeaseMinutes == 0 {
		return DefaultLeaseDuration
	}
	return time.Duration(c.LeaseMinutes) * time.Minute
}

// Validate validates configurations
func (c *DHCPConfig) Validate() error {
	for _, server := range c.DNSServers {
		ip := net.ParseIP(server)
		if ip == nil || ip.To4() == nil {
			return errors.New("invalid IPv4 address in dns-servers: " + server)
		}
	}

	return nil
}
