package etcdbackup

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type etcdBackupPodRemoveOp struct {
	apiserver *cke.Node
	finished  bool
}

// PodRemoveOp returns an Operator to Remove etcdbackup pod.
func PodRemoveOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupPodRemoveOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupPodRemoveOp) Name() string {
	return "etcdbackup-pod-remove"
}

func (o *etcdBackupPodRemoveOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return removeEtcdBackupPodCommand{o.apiserver}
}

func (o *etcdBackupPodRemoveOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type removeEtcdBackupPodCommand struct {
	apiserver *cke.Node
}

func (c removeEtcdBackupPodCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	return cs.CoreV1().Pods("kube-system").Delete(op.EtcdBackupAppName, metav1.NewDeleteOptions(60))
}

func (c removeEtcdBackupPodCommand) Command() cke.Command {
	return cke.Command{
		Name:   "remove-etcdbackup-pod",
		Target: "etcdbackup",
	}
}
