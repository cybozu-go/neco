package worker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/ingresswatcher"
)

func (o *operator) UpdateIngressWatcher(ctx context.Context, req *neco.UpdateRequest) error {
	replaced, err := o.replaceIngressWatcherFiles(ctx)
	if err != nil {
		return err
	}
	if !replaced {
		return nil
	}

	err = neco.RestartService(ctx, neco.IngressWatcher)
	if err != nil {
		return err
	}
	log.Info("ingress-watcher: updated", nil)
	return nil
}

func (o *operator) replaceIngressWatcherFiles(ctx context.Context) (bool, error) {
	buf := new(bytes.Buffer)
	err := ingresswatcher.GenerateService(buf)
	if err != nil {
		return false, err
	}
	r1, err := replaceFile(neco.ServiceFile(neco.IngressWatcher), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}
	buf.Reset()

	err = os.MkdirAll(filepath.Dir(neco.IngressWatcherConfFile), 0755)
	if err != nil {
		return false, err
	}
	err = ingresswatcher.GenerateConfBase(buf, o.mylrn)
	if err != nil {
		return false, err
	}

	r2, err := replaceFile(neco.IngressWatcherConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}
