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
	_, err := o.replaceCKEFiles(ctx, req.Servers)
	if err != nil {
		return err
	}

	if err := o.UpdateCKETemplate(ctx, req); err != nil {
		return err
	}

	err = neco.RestartService(ctx, neco.CKEService)
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
	r, err := replaceFile(neco.CKEConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}
	return r, nil
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
	defer sess.Close()
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
	}

	err = cke.UploadContents(ctx, o.localClient, o.proxyClient, req.Version, o.fetcher)
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

func (o *operator) UpdateCKETemplate(ctx context.Context, req *neco.UpdateRequest) error {
	// Check if CKE is initialized
	_, err := os.Stat(neco.CKECertFile)
	switch {
	case err == nil:
		// initialized
	case os.IsNotExist(err):
		// not initialized
		log.Info("cke: skip uploading cke-template.yml because CKE is not initialized", nil)
		return nil
	default:
		return err
	}

	// Leader election
	sess, err := concurrency.NewSession(o.ec, concurrency.WithTTL(600))
	if err != nil {
		log.Error("cke: new session is not created", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	defer sess.Close()
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

	status, err := o.storage.GetCKETemplateContentsStatus(ctx)
	if err != nil {
		if err != storage.ErrNotFound {
			return err
		}
	} else {
		if status.Version == req.Version {
			if status.Success {
				return nil
			}
			return errors.New("cke: update of cke-template.yml failed by preceding worker")
		}
	}

	err = cke.SetCKETemplate(ctx, o.storage)
	if err != nil {
		return err
	}

	ret := &neco.ContentsUpdateStatus{
		Version: req.Version,
		Success: err == nil,
	}
	err2 := o.storage.PutCKETemplateContentsStatus(ctx, ret, e.Key())
	if err2 != nil {
		log.Error("cke: failed to update cke-template.yml contents status", map[string]interface{}{
			log.FnError: err2,
		})
	}
	// 'err' is more important than 'err2'
	if err != nil {
		return err
	}

	log.Info("cke: updated cke-template.yml", nil)
	return err2
}

func (o *operator) UpdateUserResources(ctx context.Context, req *neco.UpdateRequest) error {
	// Check if CKE is alive
	isActive, err := neco.IsActiveService(ctx, neco.CKEService)
	if err != nil {
		return err
	}
	if !isActive {
		log.Info("cke: skipped uploading cke-template.yml because CKE is inactive", nil)
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
	defer sess.Close()
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

	status, err := o.storage.GetUserResourcesContentsStatus(ctx)
	if err != nil {
		if err != storage.ErrNotFound {
			return err
		}
	} else {
		if status.Version == req.Version {
			if status.Success {
				return nil
			}
			return errors.New("update of user-defined resources failed by preceding worker")
		}
	}

	err = cke.UpdateResources(ctx)
	ret := &neco.ContentsUpdateStatus{
		Version: req.Version,
		Success: err == nil,
	}
	err2 := o.storage.PutUserResourcesContentsStatus(ctx, ret, e.Key())
	if err2 != nil {
		log.Error("failed to update user-defined resources contents status", map[string]interface{}{
			log.FnError: err2,
		})
	}
	// 'err' is more important than 'err2'
	if err != nil {
		return err
	}

	log.Info("updated user-defined resources", nil)
	return err2
}
