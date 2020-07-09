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
	err := os.MkdirAll(filepath.Dir(neco.IngressWatcherConfFile), 0755)
	if err != nil {
		return false, err
	}
	buf := new(bytes.Buffer)
	err = ingresswatcher.GenerateConfBase(buf, o.mylrn)
	if err != nil {
		return false, err
	}

	return replaceFile(neco.IngressWatcherConfFile, buf.Bytes(), 0644)
}
