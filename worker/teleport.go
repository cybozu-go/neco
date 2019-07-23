package worker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/teleport"
)

func (o *operator) UpdateTeleport(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "teleport")
	if err != nil {
		return err
	}
	if need {
		err = teleport.InstallTools(ctx)
		if err != nil {
			return err
		}
		err = o.storage.RecordContainerTag(ctx, o.mylrn, "teleport")
		if err != nil {
			return err
		}
	}

	replaced, err := o.replaceTeleportFiles(ctx)
	if err != nil {
		return err
	}
	if !need && !replaced {
		return nil
	}

	err = neco.RestartService(ctx, neco.TeleportService)
	if err != nil {
		return err
	}
	log.Info("teleport: updated", nil)
	return nil
}

func (o *operator) replaceTeleportFiles(ctx context.Context) (bool, error) {
	buf := new(bytes.Buffer)
	err := teleport.GenerateService(buf)
	if err != nil {
		return false, err
	}
	r1, err := replaceFile(neco.ServiceFile(neco.TeleportService), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}
	buf.Reset()

	err = os.MkdirAll(filepath.Dir(neco.TeleportConfFile), 0755)
	if err != nil {
		return false, err
	}
	err = teleport.GenerateConfBase(buf, o.mylrn)
	if err != nil {
		return false, err
	}

	r2, err := replaceFile(neco.TeleportConfFileBase, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}
