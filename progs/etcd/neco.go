package etcd

import (
	"io/ioutil"

	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/neco"
	yaml "gopkg.in/yaml.v2"
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

	return ioutil.WriteFile(neco.NecoConfFile, data, 0644)
}
