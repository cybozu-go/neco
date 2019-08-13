package etcdbackup

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type etcdBackupSecretRemoveOp struct {
	apiserver *cke.Node
	finished  bool
}

// SecretRemoveOp returns an Operator to Remove etcdbackup certificates.
func SecretRemoveOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupSecretRemoveOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupSecretRemoveOp) Name() string {
	return "etcdbackup-secret-remove"
}

func (o *etcdBackupSecretRemoveOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return removeEtcdBackupSecretCommand{o.apiserver}
}

func (o *etcdBackupSecretRemoveOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type removeEtcdBackupSecretCommand struct {
	apiserver *cke.Node
}

func (c removeEtcdBackupSecretCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	return cs.CoreV1().Secrets("kube-system").Delete(op.EtcdBackupAppName, metav1.NewDeleteOptions(0))
}

func (c removeEtcdBackupSecretCommand) Command() cke.Command {
	return cke.Command{
		Name:   "remove-etcdbackup-secret",
		Target: "etcdbackup-secret",
	}
}
