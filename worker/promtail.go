package worker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/promtail"
)

func (o *operator) UpdatePromtail(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "promtail")
	if err != nil {
		return err
	}
	if need {
		err = o.storage.RecordContainerTag(ctx, o.mylrn, "promtail")
		if err != nil {
			return err
		}
	}

	replaced, err := o.replacePromtailFiles(ctx, o.mylrn)
	if err != nil {
		return err
	}

	if !need && !replaced {
		return nil
	}

	err = neco.RestartService(ctx, neco.PromtailService)
	if err != nil {
		return err
	}

	return nil
}

func (o *operator) replacePromtailFiles(ctx context.Context, mylrn int) (bool, error) {
	buf := new(bytes.Buffer)
	err := promtail.GenerateConf(buf, mylrn)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.PromtailConfFile), 0755)
	if err != nil {
		return false, err
	}
	r1, err := replaceFile(neco.PromtailConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	buf.Reset()
	err = promtail.GenerateService(buf, o.containerRuntime)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.ServiceFile(neco.PromtailService)), 0755)
	if err != nil {
		return false, err
	}
	r2, err := replaceFile(neco.ServiceFile(neco.PromtailService), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}
