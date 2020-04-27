package etcdbackup

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type etcdBackupConfigMapRemoveOp struct {
	apiserver *cke.Node
	finished  bool
}

// ConfigMapRemoveOp returns an Operator to Remove etcdbackup config.
func ConfigMapRemoveOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupConfigMapRemoveOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupConfigMapRemoveOp) Name() string {
	return "etcdbackup-configmap-remove"
}

func (o *etcdBackupConfigMapRemoveOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return removeEtcdBackupConfigMapCommand{o.apiserver}
}

func (o *etcdBackupConfigMapRemoveOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type removeEtcdBackupConfigMapCommand struct {
	apiserver *cke.Node
}

func (c removeEtcdBackupConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	return cs.CoreV1().ConfigMaps("kube-system").Delete(op.EtcdBackupAppName, metav1.NewDeleteOptions(0))
}

func (c removeEtcdBackupConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "remove-etcdbackup-configmap",
		Target: "etcdbackup-configmap",
	}
}
