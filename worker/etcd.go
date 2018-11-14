package worker

import (
	"context"
	"strconv"
	"strings"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

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
	return nil
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
