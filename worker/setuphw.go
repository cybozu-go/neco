package worker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/setuphw"
)

func (o *operator) UpdateSetupHW(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "setup-hw")
	if err != nil {
		return err
	}
	if need {
		err = o.storage.RecordContainerTag(ctx, o.mylrn, "setup-hw")
		if err != nil {
			return err
		}
	}

	replaced, err := o.replaceSetupHWFiles(ctx)
	if err != nil {
		return err
	}

	if need || replaced {
		err = neco.RestartService(ctx, neco.SetupHWService)
		if err != nil {
			return err
		}
		log.Info("setup-hw: updated", nil)
	}

	return nil
}

func (o *operator) replaceSetupHWFiles(ctx context.Context) (bool, error) {
	buf := new(bytes.Buffer)

	err := setuphw.GenerateService(buf)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.ServiceFile(neco.SetupHWService)), 0755)
	if err != nil {
		return false, err
	}
	return replaceFile(neco.ServiceFile(neco.SetupHWService), buf.Bytes(), 0644)
}
