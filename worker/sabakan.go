package worker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/sabakan"
)

func (o *operator) UpdateSabakan(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "sabakan")
	if err != nil {
		return err
	}
	if need {
		err = o.fetchContainer(ctx, "sabakan")
		if err != nil {
			return err
		}
		err = sabakan.InstallTools(ctx)
		if err != nil {
			return err
		}
	}

	_, err = o.replaceSabakanFiles(ctx, o.mylrn, req.Servers)
	if err != nil {
		return err
	}

	err = neco.StartService(ctx, neco.SabakanService)
	if err != nil {
		return err
	}

	log.Info("sabakan: updated", nil)
	return nil
}

func (o *operator) UpdateSabakanContents(ctx context.Context, req *neco.UpdateRequest) error {
	// Leader election

	return sabakan.UploadContents(ctx, o.localClient, o.proxyClient)
}

func (o *operator) replaceSabakanFiles(ctx context.Context, mylrn int, lrns []int) (bool, error) {
	buf := new(bytes.Buffer)
	err := sabakan.GenerateConf(buf, mylrn, lrns)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.SabakanConfFile), 0755)
	if err != nil {
		return false, err
	}
	r1, err := replaceFile(neco.SabakanConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	buf.Reset()
	err = sabakan.GenerateService(buf)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.ServiceFile("sabakan")), 0755)
	if err != nil {
		return false, err
	}
	r2, err := replaceFile(neco.ServiceFile("sabakan"), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}
