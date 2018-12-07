package worker

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/cke"
	"github.com/cybozu-go/neco/storage"
)

func (o *operator) StopCKE(ctx context.Context, req *neco.UpdateRequest) error {
	return o.stopService(ctx, neco.CKEService)
}

func (o *operator) UpdateCKE(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "cke")
	if err != nil {
		return err
	}
	if need {
		err = o.fetchContainer(ctx, "cke")
		if err != nil {
			return err
		}
		err = cke.InstallToolsCKE(ctx)
		if err != nil {
			return err
		}
	}

	need, err = o.needContainerImageUpdate(ctx, "hyperkube")
	if err != nil {
		return err
	}
	if need {
		err = o.fetchContainer(ctx, "hyperkube")
		if err != nil {
			return err
		}
		err = cke.InstallToolsHyperKube(ctx)
		if err != nil {
			return err
		}
	}

	_, err = o.replaceCKEFiles(ctx, req.Servers)
	if err != nil {
		return err
	}

	err = neco.StartService(ctx, neco.CKEService)
	if err != nil {
		return err
	}

	log.Info("cke: updated", nil)
	return nil
}

func (o *operator) replaceCKEFiles(ctx context.Context, lrns []int) (bool, error) {
	buf := new(bytes.Buffer)
	err := cke.GenerateConf(buf, lrns)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.CKEConfFile), 0755)
	if err != nil {
		return false, err
	}
	r1, err := replaceFile(neco.CKEConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	buf.Reset()
	err = cke.GenerateService(buf)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.ServiceFile("cke")), 0755)
	if err != nil {
		return false, err
	}
	r2, err := replaceFile(neco.ServiceFile(neco.CKEService), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}

func (o *operator) UpdateCKEContents(ctx context.Context, req *neco.UpdateRequest) error {
	// Check if Sabakan is alive
	isActive, err := neco.IsActiveService(ctx, neco.SabakanService)
	if err != nil {
		return err
	}
	if !isActive {
		log.Info("cke: skipped contents upload because sabakan is inactive", nil)
		return nil
	}

	// Leader election
	sess, err := concurrency.NewSession(o.ec, concurrency.WithTTL(600))
	if err != nil {
		log.Error("cke: new session is not created", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	e := concurrency.NewElection(sess, storage.KeyWorkerLeader)
	err = e.Campaign(ctx, strconv.Itoa(o.mylrn))
	if err != nil {
		log.Error("cke: cannot join a campaign", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		e.Resign(ctx)
		cancel()
	}()

	status, err := o.storage.GetCKEContentsStatus(ctx)
	if err != nil {
		if err != storage.ErrNotFound {
			return err
		}
	} else {
		if status.Version == req.Version {
			if status.Success {
				return nil
			}
			return errors.New("cke: update of contents failed by preceding worker")
		}
		if !status.Success {
			return errors.New("cke: unexpected status; success must be true if versions differ")
		}
	}

	err = cke.UploadContents(ctx, o.localClient, o.proxyClient, req.Version, o.auth)
	ret := &neco.ContentsUpdateStatus{
		Version: req.Version,
		Success: err == nil,
	}
	err2 := o.storage.PutCKEContentsStatus(ctx, ret, e.Key())
	if err2 != nil {
		log.Error("cke: failed to update contents status", map[string]interface{}{
			log.FnError: err2,
		})
	}
	// 'err' is more important than 'err2'
	if err != nil {
		return err
	}

	log.Info("cke: updated contents", nil)
	return err2
}
