package worker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/omsa"
)

func (o *operator) UpdateOMSA(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "omsa")
	if err != nil {
		return err
	}
	if need {
		err = omsa.InstallTools(ctx)
		if err != nil {
			return err
		}
		err = o.storage.RecordContainerTag(ctx, o.mylrn, "omsa")
		if err != nil {
			return err
		}
	}

	replaced, err := o.replaceOMSAFiles(ctx)
	if err != nil {
		return err
	}

	if need || replaced {
		err = neco.RestartService(ctx, neco.SetupHWService)
		if err != nil {
			return err
		}
		log.Info("omsa: updated", nil)
	}

	return nil
}

func (o *operator) replaceOMSAFiles(ctx context.Context) (bool, error) {
	buf := new(bytes.Buffer)

	err := omsa.GenerateService(buf)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.ServiceFile(neco.SetupHWService)), 0755)
	if err != nil {
		return false, err
	}
	return replaceFile(neco.ServiceFile(neco.SetupHWService), buf.Bytes(), 0644)
}
