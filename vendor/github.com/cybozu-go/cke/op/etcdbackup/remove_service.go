package etcdbackup

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type etcdBackupServiceRemoveOp struct {
	apiserver *cke.Node
	finished  bool
}

// ServiceRemoveOp returns an Operator to Remove etcdbackup service.
func ServiceRemoveOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupServiceRemoveOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupServiceRemoveOp) Name() string {
	return "etcdbackup-service-remove"
}

func (o *etcdBackupServiceRemoveOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return removeEtcdBackupServiceCommand{o.apiserver}
}

func (o *etcdBackupServiceRemoveOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type removeEtcdBackupServiceCommand struct {
	apiserver *cke.Node
}

func (c removeEtcdBackupServiceCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	return cs.CoreV1().Services("kube-system").Delete(op.EtcdBackupAppName, metav1.NewDeleteOptions(0))
}

func (c removeEtcdBackupServiceCommand) Command() cke.Command {
	return cke.Command{
		Name:   "remove-etcdbackup-service",
		Target: "etcdbackup-service",
	}
}
