package worker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcdpasswd"
)

func (o *operator) UpdateEtcdpasswd(ctx context.Context, req *neco.UpdateRequest) error {
	replaced, err := o.replaceEtcdpasswdFiles(ctx, req.Servers)
	if err != nil {
		return err
	}

	need, err := o.needDebUpdate(ctx, "etcdpasswd")
	if err != nil {
		return err
	}
	if need {
		deb, err := neco.CurrentArtifacts.FindDebianPackage("etcdpasswd")
		if err != nil {
			return err
		}
		err = InstallDebianPackage(ctx, o.proxyClient, &deb)
		if err != nil {
			return err
		}
	}

	if replaced && !need {
		err = neco.RestartService(ctx, neco.EtcdpasswdService)
		if err != nil {
			return err
		}
	}

	log.Info("etcdpasswd: updated", nil)

	return nil
}

func (o *operator) replaceEtcdpasswdFiles(ctx context.Context, lrns []int) (bool, error) {
	buf := new(bytes.Buffer)

	err := etcdpasswd.GenerateConf(buf, lrns)
	if err != nil {
		return false, err
	}

	err = os.MkdirAll(filepath.Dir(neco.EtcdpasswdConfFile), 0755)
	if err != nil {
		return false, err
	}

	return replaceFile(neco.EtcdpasswdConfFile, buf.Bytes(), 0644)
}
