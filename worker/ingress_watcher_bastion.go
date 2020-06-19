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

func (o *operator) UpdateIngressWatcherBastion(ctx context.Context, req *neco.UpdateRequest) error {
	containerName := "ingress-watcher"
	need, err := o.needContainerImageUpdate(ctx, containerName)
	if err != nil {
		return err
	}
	if need {

		err = o.storage.RecordContainerTag(ctx, o.mylrn, containerName)
		if err != nil {
			return err
		}
	}

	replaced, err := o.replaceIngressWatcherBastionFiles(ctx)
	if err != nil {
		return err
	}
	if !need && !replaced {
		return nil
	}

	err = neco.RestartService(ctx, neco.IngressWatcherBastion)
	if err != nil {
		return err
	}
	log.Info("ingress-watcher-bastion: updated", nil)
	return nil
}

func (o *operator) replaceIngressWatcherBastionFiles(ctx context.Context) (bool, error) {
	buf := new(bytes.Buffer)
	err := ingresswatcher.GenerateService(buf)
	if err != nil {
		return false, err
	}
	r1, err := replaceFile(neco.ServiceFile(neco.IngressWatcherBastion), buf.Bytes(), 0644)
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
