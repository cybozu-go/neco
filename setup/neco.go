package setup

import (
	"context"
	"io"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
	"github.com/hashicorp/vault/api"
)

func installNecoBin() error {
	_, err := os.Stat(neco.NecoBin)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	f, err := os.Open("/proc/self/exe")
	if err != nil {
		return err
	}
	defer f.Close()

	g, err := os.OpenFile(neco.NecoBin, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer g.Close()

	_, err = io.Copy(g, f)
	if err != nil {
		return err
	}

	return g.Sync()
}

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
