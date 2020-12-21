package netutil

import (
	"math/big"
	"net"
)

// IPAdd adds `val` to `ip`.
// If `ip` is IPv4 address, the value returned is IPv4, or nil when over/underflowed.
// If `ip` is IPv6 address, the value returned is IPv6, or nil when over/underflowed.
func IPAdd(ip net.IP, val int64) net.IP {
	i := big.NewInt(val)

	ipv4 := ip.To4()
	if ipv4 != nil {
		b := new(big.Int)
		b.SetBytes([]byte(ipv4))
		if i.Add(i, b).Sign() < 0 {
			return nil
		}
		res := i.Bytes()
		if len(res) < 4 {
			nres := make([]byte, 4)
			copy(nres[4-len(res):], res)
			res = nres
		}
		return net.IP(res).To4()
	}

	b := new(big.Int)
	b.SetBytes([]byte(ip))
	if i.Add(i, b).Sign() < 0 {
		return nil
	}
	res := i.Bytes()
	if len(res) < 16 {
		nres := make([]byte, 16)
		copy(nres[16-len(res):], res)
		res = nres
	}
	return net.IP(res).To16()
}

// IPDiff calculates the numeric difference between two IP addresses.
// Intuitively, the calculation is done as `ip2 - ip1`.
// `ip1` and `ip2` must be the same IP version (4 or 6).
func IPDiff(ip1, ip2 net.IP) int64 {
	if v4 := ip1.To4(); v4 != nil {
		ip1 = v4
		ip2 = ip2.To4()
	}

	b1 := new(big.Int)
	b2 := new(big.Int)
	b1.SetBytes([]byte(ip1))
	b2.SetBytes([]byte(ip2))
	return b2.Sub(b2, b1).Int64()
}
