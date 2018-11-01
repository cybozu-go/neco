package setup

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"github.com/cybozu-go/neco"
)

var templ = template.Must(template.New("etcd.config.yml").Parse(etcdConfTemplate))

func generateEtcdConf(ctx context.Context, w io.Writer, mylrn int, lrns []int) error {
	myNode0 := neco.BootNode0IP(mylrn)
	initialClusters := make([]string, len(lrns))
	for i, lrn := range lrns {
		node0 := neco.BootNode0IP(lrn)
		initialClusters[i] = fmt.Sprintf("boot-%d=https://%s:2380", lrn, node0)
	}

	err := templ.Execute(w, struct {
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
	return err
}

func setupEtcd(ctx context.Context, mylrn int, lrns []int) error {
	err := neco.RunContainer(ctx, "etcd",
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/etcd/install-tools"})
	if err != nil {
		return err
	}

	f, err := os.OpenFile(neco.EtcdConfFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	err = generateEtcdConf(ctx, f, mylrn, lrns)
	if err != nil {
		return err
	}

	return nil
}
