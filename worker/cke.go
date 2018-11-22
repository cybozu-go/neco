package worker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/cke"
)

func (o *operator) StopCKE(ctx context.Context, req *neco.UpdateRequest) error {
	return o.stopService(ctx, neco.CKEService)
}

func (o *operator) UpdateCKE(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "cke")
	if err != nil {
		return err
	}
	if need {
		err = o.fetchContainer(ctx, "cke")
		if err != nil {
			return err
		}
		err = cke.InstallToolsCKE(ctx)
		if err != nil {
			return err
		}
	}

	need, err = o.needContainerImageUpdate(ctx, "hyperkube")
	if err != nil {
		return err
	}
	if need {
		err = o.fetchContainer(ctx, "hyperkube")
		if err != nil {
			return err
		}
		err = cke.InstallToolsHyperKube(ctx)
		if err != nil {
			return err
		}
	}

	_, err = o.replaceCKEFiles(ctx, req.Servers)
	if err != nil {
		return err
	}

	err = neco.StartService(ctx, neco.CKEService)
	if err != nil {
		return err
	}

	log.Info("cke: updated", nil)
	return nil
}

func (o *operator) replaceCKEFiles(ctx context.Context, lrns []int) (bool, error) {
	buf := new(bytes.Buffer)
	err := cke.GenerateConf(buf, lrns)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.CKEConfFile), 0755)
	if err != nil {
		return false, err
	}
	r1, err := replaceFile(neco.CKEConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	buf.Reset()
	err = cke.GenerateService(buf)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.ServiceFile("cke")), 0755)
	if err != nil {
		return false, err
	}
	r2, err := replaceFile(neco.ServiceFile(neco.CKEService), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}
