package sabakan

import (
	"errors"
	"net"
	"time"

	"github.com/cybozu-go/netutil"
)

// DefaultLeaseDuration is 60 minutes.
const DefaultLeaseDuration = 60 * time.Minute

// DHCPConfig is a set of DHCP configurations.
type DHCPConfig struct {
	GatewayOffset uint     `json:"gateway-offset"`
	LeaseMinutes  uint     `json:"lease-minutes"`
	DNSServers    []string `json:"dns-servers,omitempty"`
}

// GatewayAddress returns a gateway address for the given node address
func (c *DHCPConfig) GatewayAddress(addr *net.IPNet) *net.IPNet {
	a := netutil.IP4ToInt(addr.IP.Mask(addr.Mask))
	a += uint32(c.GatewayOffset)
	return &net.IPNet{
		IP:   netutil.IntToIP4(a),
		Mask: addr.Mask,
	}
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
	if c.GatewayOffset == 0 {
		return errors.New("gateway-offset must not be zero")
	}

	for _, server := range c.DNSServers {
		ip := net.ParseIP(server)
		if ip == nil || ip.To4() == nil {
			return errors.New("invalid IPv4 address in dns-servers: " + server)
		}
	}

	return nil
}
