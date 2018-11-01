package neco

import (
	"net"

	"github.com/cybozu-go/netutil"
)

const baseNode0 = "10.69.0.0"

// BootNode0IP returns IP address of node0 for bootserver
func BootNode0IP(lrn int) net.IP {
	base := netutil.IP4ToInt(net.ParseIP(baseNode0))
	return netutil.IntToIP4(base + 192*uint32(lrn) + 3)
}
