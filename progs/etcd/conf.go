package etcd

import (
	"fmt"
	"io"
	"strings"

	"github.com/cybozu-go/neco"
	"go.etcd.io/etcd/etcdserver/etcdserverpb"
)

// GenerateConf generates etcd.conf.yml from template.
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

// GenerateConfForAdd generates etcd.conf.yml from template to add a new member.
func GenerateConfForAdd(w io.Writer, mylrn int, members []*etcdserverpb.Member) error {
	myNode0 := neco.BootNode0IP(mylrn)
	var initialClusters []string
	for _, member := range members {
		name := member.Name
		if name == "" {
			name = fmt.Sprintf("boot-%d", mylrn)
		}
		for _, u := range member.PeerURLs {
			initialClusters = append(initialClusters, fmt.Sprintf("%s=%s", name, u))
		}
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
		InitialClusterState:      "existing",
	})
}
