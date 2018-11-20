package worker

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/serf"
)

func (o *operator) UpdateSerf(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "serf")
	if err != nil {
		return err
	}
	if need {
		err = o.fetchContainer(ctx, "serf")
		if err != nil {
			return err
		}
		err = serf.InstallTools(ctx)
		if err != nil {
			return err
		}
	}
	_, err = o.replaceSerfFiles(ctx, req.Servers)
	if err != nil {
		return err
	}
	err = neco.RestartService(ctx, neco.SerfService)
	if err != nil {
		return err
	}
	log.Info("serf: updated", nil)
	return nil
}

func (o *operator) replaceSerfFiles(ctx context.Context, lrns []int) (bool, error) {
	buf := new(bytes.Buffer)
	err := serf.GenerateService(buf)
	if err != nil {
		return false, err
	}

	r1, err := replaceFile(neco.ServiceFile(neco.SerfService), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	buf.Reset()
	osVer, err := serf.GetOSVersion()
	if err != nil {
		return false, err
	}
	serial, err := exec.Command("cat", "/sys/class/dmi/id/product_serial").Output()
	if err != nil {
		return false, err
	}
	err = serf.GenerateConf(buf, lrns, osVer, string(serial))
	if err != nil {
		return false, err
	}

	r2, err := replaceFile(neco.SerfConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}
