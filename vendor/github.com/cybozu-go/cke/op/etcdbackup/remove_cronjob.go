package etcdbackup

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type etcdBackupCronJobRemoveOp struct {
	apiserver *cke.Node
	finished  bool
}

// CronJobRemoveOp returns an Operator to Remove etcdbackup cron job.
func CronJobRemoveOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupCronJobRemoveOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupCronJobRemoveOp) Name() string {
	return "etcdbackup-job-remove"
}

func (o *etcdBackupCronJobRemoveOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return removeEtcdBackupCronJobCommand{o.apiserver}
}

func (o *etcdBackupCronJobRemoveOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type removeEtcdBackupCronJobCommand struct {
	apiserver *cke.Node
}

func (c removeEtcdBackupCronJobCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	return cs.BatchV1beta1().CronJobs("kube-system").Delete(op.EtcdBackupAppName, metav1.NewDeleteOptions(60))
}

func (c removeEtcdBackupCronJobCommand) Command() cke.Command {
	return cke.Command{
		Name:   "remove-etcdbackup-job",
		Target: "etcdbackup",
	}
}
