package worker

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/sabakan"
	"github.com/cybozu-go/neco/storage"
)

func (o *operator) UpdateSabakan(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "sabakan")
	if err != nil {
		return err
	}
	if need {
		err = sabakan.InstallTools(ctx)
		if err != nil {
			return err
		}
		err = o.storage.RecordContainerTag(ctx, o.mylrn, "sabakan")
		if err != nil {
			return err
		}
	}

	_, err = o.replaceSabakanFiles(ctx, o.mylrn, req.Servers)
	if err != nil {
		return err
	}

	err = neco.RestartService(ctx, neco.SabakanService)
	if err != nil {
		return err
	}

	_, err = os.Stat(neco.SabakanKeyFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("sabakan: updated", nil)
			return nil
		}
		return err
	}

	hreq, err := http.NewRequest(http.MethodGet, neco.SabakanLocalEndpoint, nil)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	for {
		hreq := hreq.WithContext(ctx)
		resp, err := o.localClient.Do(hreq)
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
			}
			continue
		}
		resp.Body.Close()
		log.Info("sabakan: updated", nil)
		return nil
	}
}

func (o *operator) UpdateSabakanContents(ctx context.Context, req *neco.UpdateRequest) error {
	// Check if Sabakan is alive
	isActive, err := neco.IsActiveService(ctx, neco.SabakanService)
	if err != nil {
		return err
	}
	if !isActive {
		log.Info("sabakan: skipped uploading contents because sabakan is inactive", nil)
		return nil
	}

	auth, err := o.getDockerAuth(ctx, o.storage)
	if err == storage.ErrNotFound {
		log.Info("sabakan: skipped uploading contents because Quay auth is not set", nil)
		return nil
	}
	if err != nil {
		return err
	}

	// Leader election
	sess, err := concurrency.NewSession(o.ec, concurrency.WithTTL(600))
	if err != nil {
		log.Error("sabakan: new session is not created", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	e := concurrency.NewElection(sess, storage.KeyWorkerLeader)
	err = e.Campaign(ctx, strconv.Itoa(o.mylrn))
	if err != nil {
		log.Error("sabakan: cannot join a campaign", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		e.Resign(ctx)
		cancel()
	}()

	status, err := o.storage.GetSabakanContentsStatus(ctx)
	if err != nil {
		if err != storage.ErrNotFound {
			return err
		}
	} else {
		if status.Version == req.Version {
			if status.Success {
				return nil
			}
			return errors.New("update of Sabakan contents failed by preceding worker")
		}
		if !status.Success {
			return errors.New("unexpected status; success must be true if versions differ")
		}
	}

	err = sabakan.UploadContents(ctx, o.localClient, o.proxyClient, req.Version, auth, o.storage)
	ret := &neco.ContentsUpdateStatus{
		Version: req.Version,
		Success: err == nil,
	}
	err2 := o.storage.PutSabakanContentsStatus(ctx, ret, e.Key())
	if err2 != nil {
		log.Error("sabakan: failed to update Sabakan contents status", map[string]interface{}{
			log.FnError: err2,
		})
	}
	// 'err' is more important than 'err2'
	if err != nil {
		return err
	}

	log.Info("sabakan: updated contents", nil)
	return err2
}

func (o *operator) replaceSabakanFiles(ctx context.Context, mylrn int, lrns []int) (bool, error) {
	buf := new(bytes.Buffer)
	err := sabakan.GenerateConf(buf, mylrn, lrns)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.SabakanConfFile), 0755)
	if err != nil {
		return false, err
	}
	r1, err := replaceFile(neco.SabakanConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	buf.Reset()
	err = sabakan.GenerateService(buf)
	if err != nil {
		return false, err
	}
	err = os.MkdirAll(filepath.Dir(neco.ServiceFile("sabakan")), 0755)
	if err != nil {
		return false, err
	}
	r2, err := replaceFile(neco.ServiceFile("sabakan"), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return r1 || r2, nil
}
