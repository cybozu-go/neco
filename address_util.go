package neco

import (
	"fmt"
	"net"

	"github.com/cybozu-go/netutil"
)

const baseNode0 = "10.69.0.0"

// BootNode0IP returns IP address of node0 for bootserver
func BootNode0IP(lrn int) net.IP {
	return netutil.IPAdd(net.ParseIP(baseNode0), int64(192*lrn+3))
}

// EtcdEndpoints returns a list of etcd endpoints for LRNs.
func EtcdEndpoints(lrns []int) []string {
	l := make([]string, len(lrns))
	for i, lrn := range lrns {
		l[i] = fmt.Sprintf("https://%s:2379", BootNode0IP(lrn).String())
	}
	return l
}
