package neco

import (
	"os"

	"github.com/cybozu-go/etcdutil"
	clientv3 "go.etcd.io/etcd/client/v3"
	"sigs.k8s.io/yaml"
)

// EtcdClient returns etcd client for Neco tools.
func EtcdClient() (*clientv3.Client, error) {
	data, err := os.ReadFile(NecoConfFile)
	if err != nil {
		return nil, err
	}

	cfg := etcdutil.NewConfig(NecoPrefix)
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return etcdutil.NewClient(cfg)
}
