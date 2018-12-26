package worker

import (
	"bytes"
	"context"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/vault"
)

func (o *operator) StopVault(ctx context.Context, req *neco.UpdateRequest) error {
	return o.stopService(ctx, neco.VaultService)
}

func (o *operator) UpdateVault(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "vault")
	if err != nil {
		return err
	}
	if need {
		err = vault.InstallTools(ctx)
		if err != nil {
			return err
		}
		err = o.storage.RecordContainerTag(ctx, o.mylrn, "vault")
		if err != nil {
			return err
		}
	}

	_, err = o.replaceVaultFiles(ctx, req.Servers)
	if err != nil {
		return err
	}

	err = neco.StartService(ctx, neco.VaultService)
	if err != nil {
		return err
	}

	// wait for vault leader election
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
	}

	log.Info("vault: updated", nil)
	return nil
}

func (o *operator) replaceVaultFiles(ctx context.Context, lrns []int) (bool, error) {
	buf := new(bytes.Buffer)
	err := vault.GenerateService(buf)
	if err != nil {
		return false, err
	}

	r1, err := replaceFile(neco.ServiceFile(neco.VaultService), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	buf.Reset()
	err = vault.GenerateConf(buf, o.mylrn, lrns)
	if err != nil {
		return false, err
	}

	r2, err := replaceFile(neco.VaultConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}
