package etcd

import "github.com/cybozu-go/cke"

type etcdWaitClusterOp struct {
	endpoints []string
	executed  bool
}

// WaitClusterOp returns an Operator to wait until etcd cluster becomes healthy
func WaitClusterOp(nodes []*cke.Node) cke.Operator {
	return &etcdWaitClusterOp{
		endpoints: etcdEndpoints(nodes),
	}
}

func (o *etcdWaitClusterOp) Name() string {
	return "etcd-wait-cluster"
}

func (o *etcdWaitClusterOp) NextCommand() cke.Commander {
	if o.executed {
		return nil
	}
	o.executed = true

	return waitEtcdSyncCommand{o.endpoints, false}
}

func (o *etcdWaitClusterOp) Targets() []string {
	return o.endpoints
}
