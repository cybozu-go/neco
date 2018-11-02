package setup

import (
	"context"
	"io/ioutil"

	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/hashicorp/vault/api"
	yaml "gopkg.in/yaml.v2"
)

func setupNecoFiles(ctx context.Context, vc *api.Client, lrns []int) error {
	secret, err := vc.Logical().Write(neco.CAEtcdClient+"/issue/system", map[string]interface{}{
		"common_name":          "neco",
		"exclude_cn_from_sans": true,
	})
	err = dumpCertFiles(secret, "", neco.NecoCertFile, neco.NecoKeyFile)
	if err != nil {
		return err
	}

	cfg := etcdutil.NewConfig(neco.NecoPrefix)
	cfg.Endpoints = neco.EtcdEndpoints(lrns)
	cfg.TLSCertFile = neco.NecoCertFile
	cfg.TLSKeyFile = neco.NecoKeyFile

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(neco.NecoConfFile, data, 0644)
	if err != nil {
		return err
	}

	log.Info("neco: generated files", nil)

	return nil
}
