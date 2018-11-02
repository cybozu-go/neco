package setup

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
	"github.com/hashicorp/vault/api"
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

	err = etcd.UpdateNecoConfig(lrns)
	if err != nil {
		return err
	}

	log.Info("neco: generated files", nil)

	return nil
}
