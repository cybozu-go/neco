package etcd

import (
	"os"

	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/neco"
	"sigs.k8s.io/yaml"
)

// UpdateNecoConfig updates /etc/neco/config.yml.
func UpdateNecoConfig(lrns []int) error {
	cfg := etcdutil.NewConfig(neco.NecoPrefix)
	cfg.Endpoints = neco.EtcdEndpoints(lrns)
	cfg.TLSCertFile = neco.NecoCertFile
	cfg.TLSKeyFile = neco.NecoKeyFile

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(neco.NecoConfFile, data, 0644)
}
