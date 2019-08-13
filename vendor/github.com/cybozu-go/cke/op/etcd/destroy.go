package etcd

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type destroyMemberOp struct {
	endpoints []string
	targets   []*cke.Node
	ids       []uint64
	params    cke.EtcdParams
	step      int
}

// DestroyMemberOp returns an Operator to remove and destroy a member.
func DestroyMemberOp(cp []*cke.Node, targets []*cke.Node, ids []uint64) cke.Operator {
	return &destroyMemberOp{
		endpoints: etcdEndpoints(cp),
		targets:   targets,
		ids:       ids,
	}
}

func (o *destroyMemberOp) Name() string {
	return "etcd-destroy-member"
}

func (o *destroyMemberOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return removeMemberCommand{o.endpoints, o.ids}
	case 1:
		o.step++
		return common.KillContainersCommand(o.targets, op.EtcdContainerName)
	case 2:
		o.step++
		return common.VolumeRemoveCommand(o.targets, op.EtcdVolumeName(o.params))
	case 3:
		o.step++
		return waitEtcdSyncCommand{o.endpoints, false}
	}
	return nil
}

func (o *destroyMemberOp) Targets() []string {
	ips := make([]string, len(o.targets))
	for i, n := range o.targets {
		ips[i] = n.Address
	}
	return ips
}
