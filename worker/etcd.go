package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
	"github.com/cybozu-go/neco/storage"
)

const etcdAddTimeout = 10 * time.Minute

func (o *operator) UpdateEtcd(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "etcd")
	if err != nil {
		return err
	}
	if need {
		err = o.fetchContainer(ctx, "etcd")
		if err != nil {
			return err
		}
	}

	// leader election
	session, err := concurrency.NewSession(o.ec, concurrency.WithTTL(10))
	if err != nil {
		return err
	}
	e := concurrency.NewElection(session, storage.KeyWorkerLeader)
	err = e.Campaign(ctx, strconv.Itoa(o.mylrn))
	if err != nil {
		return err
	}
	defer e.Resign(ctx)

	mlr, err := o.ec.MemberList(ctx)
	if err != nil {
		log.Error("failed to list members", map[string]interface{}{log.FnError: err.Error()})
		return err
	}
	err = o.removeEtcdMembers(ctx, mlr, req)
	if err != nil {
		log.Error("failed to remove members", map[string]interface{}{log.FnError: err.Error()})
		return err
	}

	if !isMember(mlr.Members, o.mylrn) {
		// wait for several seconds to satisfy etcd server check
		// https://github.com/etcd-io/etcd/blob/fb674833c21e729fe87fff4addcf93b2aa4df9df/etcdserver/server.go#L1562
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}

		err = o.addEtcdMember(ctx)
		if err != nil {
			log.Error("failed to add a member", map[string]interface{}{log.FnError: err.Error()})
			return err
		}
	}

	replaced, err := o.replaceEtcdFiles(ctx, req.Servers)
	if err != nil {
		return err
	}

	err = etcd.UpdateNecoConfig(req.Servers)
	if err != nil {
		return err
	}

	if need || replaced {
		err = neco.StopService(ctx, neco.EtcdService)
		if err != nil {
			return err
		}

		err = neco.StartService(ctx, neco.EtcdService)
		if err != nil {
			return err
		}

		ec, err := etcd.WaitEtcdForVault(ctx)
		if err != nil {
			return err
		}
		resp, err := ec.Get(ctx, "/")
		if err != nil {
			return err
		}
		err = waitEtcdSync(ctx, ec, resp.Header.Revision)
		if err != nil {
			return err
		}
		ec.Close()
	}

	return nil
}

func isMember(members []*etcdserverpb.Member, lrn int) bool {
	for _, member := range members {
		if member.Name == fmt.Sprintf("boot-%d", lrn) {
			return true
		}
	}
	return false
}

func (o *operator) removeEtcdMembers(ctx context.Context, mlr *clientv3.MemberListResponse, req *neco.UpdateRequest) error {
	var toRemove []*etcdserverpb.Member

OUTER:
	for _, member := range mlr.Members {
		if !strings.HasPrefix(member.Name, "boot-") {
			log.Info("removing etcd member", map[string]interface{}{
				"name": member.Name,
			})
			toRemove = append(toRemove, member)
			continue
		}
		lrn, err := strconv.Atoi(member.Name[5:])
		if err != nil {
			log.Info("removing etcd member", map[string]interface{}{
				"name": member.Name,
			})
			toRemove = append(toRemove, member)
			continue
		}
		for _, server := range req.Servers {
			if lrn == server {
				continue OUTER
			}
		}
		log.Info("removing etcd member", map[string]interface{}{
			"name": member.Name,
		})
		toRemove = append(toRemove, member)
	}
	for _, member := range toRemove {
		_, err := o.ec.MemberRemove(ctx, member.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *operator) addEtcdMember(ctx context.Context) error {
	node0 := neco.BootNode0IP(o.mylrn)
	peerURL := fmt.Sprintf("https://%s:2380", node0.String())
	resp, err := o.ec.MemberAdd(ctx, []string{peerURL})
	if err != nil {
		return err
	}

	ec, err := etcd.Setup(ctx, func(w io.Writer) error {
		return etcd.GenerateConfForAdd(w, o.mylrn, resp.Members)
	})
	if err != nil {
		return err
	}
	defer ec.Close()

	err = o.storage.RecordContainerTag(ctx, o.mylrn, "etcd")
	if err != nil {
		return err
	}

	return waitEtcdSync(ctx, ec, resp.Header.Revision)
}

func waitEtcdSync(ctx context.Context, ec *clientv3.Client, rev int64) error {
	deadline := time.Now().Add(etcdAddTimeout)
	for {
		if time.Now().After(deadline) {
			return errors.New("etcd does not synchronize")
		}
		gr, err := ec.Get(ctx, "health")
		if err == nil && gr.Header.Revision >= rev {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func (o *operator) replaceEtcdFiles(ctx context.Context, lrns []int) (bool, error) {
	buf := new(bytes.Buffer)
	err := etcd.GenerateService(buf)
	if err != nil {
		return false, err
	}

	r1, err := replaceFile(neco.ServiceFile(neco.EtcdService), buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	buf.Reset()
	err = etcd.GenerateConf(buf, o.mylrn, lrns)
	if err != nil {
		return false, err
	}

	r2, err := replaceFile(neco.EtcdConfFile, buf.Bytes(), 0644)
	if err != nil {
		return false, err
	}

	return (r1 || r2), nil
}
