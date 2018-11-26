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
	hw, err := neco.DetectHardware()
	if err != nil {
		return err
	}
	if hw != neco.HWTypeDell {
		return nil
	}

	need, err := o.needContainerImageUpdate(ctx, "omsa")
	if err != nil {
		return err
	}
	if need {
		err = o.fetchContainer(ctx, "omsa")
		if err != nil {
			return err
		}
		err = omsa.InstallTools(ctx)
		if err != nil {
			return err
		}
	}

	replaced, err := o.replaceOMSAFiles(ctx)
	if err != nil {
		return err
	}

	if need || replaced {
		err = neco.RestartService(ctx, neco.OMSAService)
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
	err = os.MkdirAll(filepath.Dir(neco.ServiceFile(neco.OMSAService)), 0755)
	if err != nil {
		return false, err
	}
	return replaceFile(neco.ServiceFile(neco.OMSAService), buf.Bytes(), 0644)
}
