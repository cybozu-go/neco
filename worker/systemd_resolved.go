package worker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/systemdresolved"
	"github.com/cybozu-go/neco/storage"
)

func (o *operator) UpdateSystemdResolved(ctx context.Context, req *neco.UpdateRequest) error {
	replaced, err := o.replaceSystemdResolvedFiles(ctx)
	if err == storage.ErrNotFound {
		log.Info("systemd-resolved: dns config not found", nil)
		return nil
	}
	if err != nil {
		return err
	}
	if !replaced {
		return nil
	}

	err = neco.RestartService(ctx, neco.SystemdResolved)
	if err != nil {
		return err
	}
	log.Info("systemd-resolved: updated", nil)
	return nil
}

func (o *operator) replaceSystemdResolvedFiles(ctx context.Context) (bool, error) {
	buf := new(bytes.Buffer)

	err := os.MkdirAll(filepath.Dir(neco.SystemdResolvedConfFile), 0755)
	if err != nil {
		return false, err
	}
	dnsAddress, err := o.storage.GetDNSConfig(ctx)
	if err != nil {
		return false, err
	}
	err = systemdresolved.GenerateConfBase(buf, dnsAddress)
	if err != nil {
		return false, err
	}

	r1, err := replaceFile(neco.SystemdResolvedConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1, nil
}
