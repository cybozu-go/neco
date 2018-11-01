package etcd

import (
	"fmt"
	"io"
	"strings"

	"github.com/cybozu-go/neco"
)

// GenerateConf genenates etcd.conf.yml from template.
func GenerateConf(w io.Writer, mylrn int, lrns []int) error {
	myNode0 := neco.BootNode0IP(mylrn)
	initialClusters := make([]string, len(lrns))
	for i, lrn := range lrns {
		node0 := neco.BootNode0IP(lrn)
		initialClusters[i] = fmt.Sprintf("boot-%d=https://%s:2380", lrn, node0)
	}

	return confTmpl.Execute(w, struct {
		LRN                      int
		InitialAdvertisePeerURLs string
		AdvertiseClientURLs      string
		InitialCluster           string
		InitialClusterState      string
	}{
		LRN:                      mylrn,
		InitialAdvertisePeerURLs: fmt.Sprintf("https://%s:2380", myNode0),
		AdvertiseClientURLs:      fmt.Sprintf("https://%s:2379", myNode0),
		InitialCluster:           strings.Join(initialClusters, ","),
		InitialClusterState:      "new",
	})
}
