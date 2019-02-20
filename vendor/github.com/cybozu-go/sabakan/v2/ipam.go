package sabakan

import (
	"errors"
	"net"

	"github.com/cybozu-go/netutil"
)

// IPAMConfig is a set of IPAM configurations.
type IPAMConfig struct {
	MaxNodesInRack    uint   `json:"max-nodes-in-rack"`
	NodeIPv4Pool      string `json:"node-ipv4-pool"`
	NodeIPv4Offset    string `json:"node-ipv4-offset,omitempty"`
	NodeRangeSize     uint   `json:"node-ipv4-range-size"`
	NodeRangeMask     uint   `json:"node-ipv4-range-mask"`
	NodeIPPerNode     uint   `json:"node-ip-per-node"`
	NodeIndexOffset   uint   `json:"node-index-offset"`
	NodeGatewayOffset uint   `json:"node-gateway-offset"`

	BMCIPv4Pool      string `json:"bmc-ipv4-pool"`
	BMCIPv4Offset    string `json:"bmc-ipv4-offset,omitempty"`
	BMCRangeSize     uint   `json:"bmc-ipv4-range-size"`
	BMCRangeMask     uint   `json:"bmc-ipv4-range-mask"`
	BMCGatewayOffset uint   `json:"bmc-ipv4-gateway-offset"`
}

// Validate validates configurations
func (c *IPAMConfig) Validate() error {
	if c.MaxNodesInRack == 0 {
		return errors.New("max-nodes-in-rack must not be zero")
	}

	ip, ipNet, err := net.ParseCIDR(c.NodeIPv4Pool)
	if err != nil {
		return errors.New("invalid node-ipv4-pool")
	}
	if !ip.Equal(ipNet.IP) {
		return errors.New("host part of node-ipv4-pool must be cleared")
	}
	if len(c.NodeIPv4Offset) > 0 && net.ParseIP(c.NodeIPv4Offset) == nil {
		return errors.New("invalid node-ipv4-offset")
	}
	if c.NodeRangeSize == 0 {
		return errors.New("node-ipv4-range-size must not be zero")
	}
	if c.NodeRangeMask < 8 || 32 < c.NodeRangeMask {
		return errors.New("invalid node-ipv4-range-mask")
	}
	if c.NodeIPPerNode == 0 {
		return errors.New("node-ip-per-node must not be zero")
	}
	if c.NodeIndexOffset == 0 {
		return errors.New("node-index-offset must not be zero")
	}
	if c.NodeGatewayOffset == 0 {
		return errors.New("node-gateway-offset must not be zero")
	}

	ip, ipNet, err = net.ParseCIDR(c.BMCIPv4Pool)
	if err != nil {
		return errors.New("invalid bmc-ipv4-pool")
	}
	if !ip.Equal(ipNet.IP) {
		return errors.New("host part of bmc-ipv4-pool must be cleared")
	}
	if len(c.BMCIPv4Offset) > 0 && net.ParseIP(c.BMCIPv4Offset) == nil {
		return errors.New("invalid bmc-ipv4-offset")
	}
	if c.BMCRangeSize == 0 {
		return errors.New("bmc-ipv4-range-size must not be zero")
	}
	if c.BMCRangeMask < 8 || 32 < c.BMCRangeMask {
		return errors.New("invalid bmc-ipv4-range-mask")
	}
	if c.BMCGatewayOffset == 0 {
		return errors.New("bmc-ipv4-gateway-offset must not be zero")
	}

	return nil
}

// GatewayAddress returns a gateway address for the given node address
func (c *IPAMConfig) GatewayAddress(addr *net.IPNet) *net.IPNet {
	a := netutil.IP4ToInt(addr.IP.Mask(addr.Mask))
	a += uint32(c.NodeGatewayOffset)
	return &net.IPNet{
		IP:   netutil.IntToIP4(a),
		Mask: addr.Mask,
	}
}

// GenerateIP generates IP addresses for a machine.
// Generated IP addresses are stored in mc.
func (c *IPAMConfig) GenerateIP(mc *Machine) {
	// IP addresses are calculated as follows (LRN=Logical Rack Number):
	// node0: INET_NTOA(INET_ATON(NodeIPv4Pool) + INET_ATON(NodeIPv4Offset) + (2^NodeRangeSize * NodeIPPerNode * LRN) + index-in-rack)
	// node1: INET_NTOA(INET_ATON(NodeIPv4Pool) + INET_ATON(NodeIPv4Offset) + (2^NodeRangeSize * NodeIPPerNode * LRN) + index-in-rack + 2^NodeRangeSize)
	// node2: INET_NTOA(INET_ATON(NodeIPv4Pool) + INET_ATON(NodeIPv4Offset) + (2^NodeRangeSize * NodeIPPerNode * LRN) + index-in-rack + 2^NodeRangeSize * 2)
	// BMC: INET_NTOA(INET_ATON(BMCIPv4Pool) + INET_ATON(BMCIPv4Offset) + (2^BMCRangeSize * LRN) + index-in-rack)

	calc := func(pool, offset string, shift, numip, lrn, idx uint) []net.IP {
		result := make([]net.IP, numip)

		poolIP, _, _ := net.ParseCIDR(pool)
		noffset := uint32(0)
		if len(offset) > 0 {
			noffset = netutil.IP4ToInt(net.ParseIP(offset))
		}
		a := netutil.IP4ToInt(poolIP) + noffset
		su := uint(1) << shift
		for i := uint(0); i < numip; i++ {
			ip := netutil.IntToIP4(a + uint32(su*numip*lrn+idx+i*su))
			result[i] = ip
		}
		return result
	}

	lrn := mc.Spec.Rack
	idx := mc.Spec.IndexInRack

	ips := calc(c.NodeIPv4Pool, c.NodeIPv4Offset, c.NodeRangeSize, c.NodeIPPerNode, lrn, idx)
	strIPs := make([]string, len(ips))
	nics := make([]NICConfig, len(ips))
	mask := net.CIDRMask(int(c.NodeRangeMask), 32)
	strMask := net.IP(mask).String()
	for i, p := range ips {
		strP := p.String()
		strIPs[i] = strP
		nics[i].Address = strP
		nics[i].Netmask = strMask
		nics[i].MaskBits = int(c.NodeRangeMask)
		gw := c.GatewayAddress(&net.IPNet{IP: p, Mask: mask})
		nics[i].Gateway = gw.IP.String()
	}
	mc.Spec.IPv4 = strIPs
	mc.Spec.IPv6 = nil
	mc.Info.Network.IPv4 = nics

	bmcIPs := calc(c.BMCIPv4Pool, c.BMCIPv4Offset, c.BMCRangeSize, 1, lrn, idx)
	mc.Spec.BMC.IPv4 = bmcIPs[0].String()
	mc.Spec.BMC.IPv6 = ""
	bmcMask := net.CIDRMask(int(c.BMCRangeMask), 32)
	mc.Info.BMC.IPv4.Address = mc.Spec.BMC.IPv4
	mc.Info.BMC.IPv4.Netmask = net.IP(bmcMask).String()
	mc.Info.BMC.IPv4.MaskBits = int(c.BMCRangeMask)
	bmcGW := netutil.IntToIP4(netutil.IP4ToInt(bmcIPs[0].Mask(bmcMask)) + uint32(c.BMCGatewayOffset))
	mc.Info.BMC.IPv4.Gateway = bmcGW.String()
}

// LeaseRange is a range of IP addresses for DHCP lease.
type LeaseRange struct {
	BeginAddress net.IP
	Count        int
	key          string
}

// IP returns n-th IP address in the range.
func (l *LeaseRange) IP(n int) net.IP {
	naddr := netutil.IP4ToInt(l.BeginAddress) + uint32(n)
	return netutil.IntToIP4(naddr)
}

// Key return key string.
func (l *LeaseRange) Key() string {
	if len(l.key) == 0 {
		l.key = l.BeginAddress.String()
	}
	return l.key
}

// LeaseRange returns a LeaseRange for the interface that receives DHCP requests.
// If no range can be assigned, this returns nil.
func (c *IPAMConfig) LeaseRange(ifaddr net.IP) *LeaseRange {
	ip1, _, _ := net.ParseCIDR(c.NodeIPv4Pool)
	noffset1 := uint32(0)
	if len(c.NodeIPv4Offset) > 0 {
		noffset1 = netutil.IP4ToInt(net.ParseIP(c.NodeIPv4Offset))
	}
	nip1 := netutil.IP4ToInt(ip1) + noffset1
	nip2 := netutil.IP4ToInt(ifaddr)
	if nip2 <= nip1 {
		return nil
	}

	// Given these configurations,
	//   MaxNodesInRack  = 28
	//   NodeRangeSize   = 6
	//   NodeIndexOffset = 3
	//
	// The lease range will start at offset 32, and ends at 62 (64 - 1 - 1).
	// Therefore the available lease IP address count is 31.

	rangeSize := uint32(1 << c.NodeRangeSize)
	offset := uint32(c.NodeIndexOffset + c.MaxNodesInRack + 1)

	ranges := (nip2 - nip1) / rangeSize
	rangeStart := nip1 + rangeSize*ranges + uint32(c.NodeIndexOffset+c.MaxNodesInRack+1)
	startIP := netutil.IntToIP4(rangeStart)
	count := (rangeSize - 2) - offset + 1
	return &LeaseRange{
		BeginAddress: startIP,
		Count:        int(count),
	}
}
