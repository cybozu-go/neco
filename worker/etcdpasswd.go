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
	if err := etcdpasswd.InstallSshdConf(); err != nil {
		return err
	}

	if err := neco.RestartService(ctx, "ssh"); err != nil {
		return err
	}

	if err := etcdpasswd.InstallSudoers(); err != nil {
		return err
	}

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
		err = InstallDebianPackage(ctx, o.proxyClient, o.ghClient, &deb, false, nil)
		if err != nil {
			return err
		}
		err = o.storage.RecordDebVersion(ctx, o.mylrn, "etcdpasswd")
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

	err := etcdpasswd.GenerateSystemdDropIn(buf)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.EtcdpasswdDropIn), 0755)
	if err != nil {
		return false, err
	}
	r1, err := replaceFile(neco.EtcdpasswdDropIn, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	buf.Reset()
	err = etcdpasswd.GenerateConf(buf, lrns)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.EtcdpasswdConfFile), 0755)
	if err != nil {
		return false, err
	}
	r2, err := replaceFile(neco.EtcdpasswdConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}
