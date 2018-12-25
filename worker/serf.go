package worker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/serf"
	"github.com/cybozu-go/neco/storage"
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
		err = o.storage.RecordContainerTag(ctx, o.mylrn, "serf")
		if err != nil {
			return err
		}
	}
	replaced, err := o.replaceSerfFiles(ctx, req.Servers)
	if err != nil {
		return err
	}
	if !need && !replaced {
		return nil
	}

	sess, err := concurrency.NewSession(o.ec, concurrency.WithTTL(10))
	if err != nil {
		log.Error("serf: new session is not created", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	e := concurrency.NewElection(sess, storage.KeyWorkerLeader)
	err = e.Campaign(ctx, strconv.Itoa(o.mylrn))
	if err != nil {
		log.Error("serf: cannot join a campaign", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		e.Resign(ctx)
		cancel()

		log.Info("serf: updated", nil)
	}()

	err = neco.RestartService(ctx, neco.SerfService)
	if err != nil {
		return err
	}

	return nil
}

func (o *operator) replaceSerfFiles(ctx context.Context, lrns []int) (bool, error) {
	buf := new(bytes.Buffer)
	err := serf.GenerateService(buf)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.SerfConfFile), 0755)
	if err != nil {
		return false, err
	}
	r1, err := replaceFile(neco.ServiceFile(neco.SerfService), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	buf.Reset()
	os, err := serf.GetOSVersionID()
	if err != nil {
		return false, err
	}
	serial, err := serf.GetSerial()
	if err != nil {
		return false, err
	}
	err = serf.GenerateConf(buf, lrns, os, serial)
	if err != nil {
		return false, err
	}

	r2, err := replaceFile(neco.SerfConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}
