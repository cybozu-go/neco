package worker

import (
	"context"
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
	"github.com/cybozu-go/neco/setup"
	"github.com/cybozu-go/neco/storage"
	"github.com/pkg/errors"
)

const etcdAddTimeout = 10 * time.Minute

func (o *operator) UpdateEtcd(ctx context.Context, req *neco.UpdateRequest) error {
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
	//leaderKey := e.Key()
	mlr, err := o.ec.MemberList(ctx)
	if err != nil {
		return err
	}
	err = o.removeEtcdMembers(ctx, mlr, req)
	if err != nil {
		return err
	}

	if !isMember(mlr.Members, o.mylrn) {
		err = o.addEtcdMember(ctx)
		if err != nil {
			return err
		}
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

	ec, err := setup.SetupEtcd(ctx, func(w io.Writer) error {
		return etcd.GenerateConfForAdd(w, o.mylrn, resp.Members)
	})
	if err != nil {
		return err
	}
	defer ec.Close()

	deadline := time.Now().Add(etcdAddTimeout)
	for {
		if time.Now().After(deadline) {
			return errors.New("etcd add timed out")
		}
		gr, err := ec.Get(ctx, "health")
		if err == nil && gr.Header.Revision >= resp.Header.Revision {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}
