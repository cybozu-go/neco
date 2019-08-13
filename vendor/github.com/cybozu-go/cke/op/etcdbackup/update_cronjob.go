package etcdbackup

import (
	"bytes"
	"context"

	"github.com/cybozu-go/cke"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type etcdBackupCronJobUpdateOp struct {
	apiserver *cke.Node
	schedule  string
	finished  bool
}

// CronJobUpdateOp returns an Operator to Update etcdbackup cron job.
func CronJobUpdateOp(apiserver *cke.Node, schedule string) cke.Operator {
	return &etcdBackupCronJobUpdateOp{
		apiserver: apiserver,
		schedule:  schedule,
	}
}

func (o *etcdBackupCronJobUpdateOp) Name() string {
	return "etcdbackup-job-update"
}

func (o *etcdBackupCronJobUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateEtcdBackupCronJobCommand{o.apiserver, o.schedule}
}

func (o *etcdBackupCronJobUpdateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type updateEtcdBackupCronJobCommand struct {
	apiserver *cke.Node
	schedule  string
}

func (c updateEtcdBackupCronJobCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = cronJobTemplate.Execute(buf, struct {
		Schedule string
	}{
		Schedule: c.schedule,
	})
	if err != nil {
		return err
	}

	cronJob := new(batchv1beta1.CronJob)
	err = k8sYaml.NewYAMLToJSONDecoder(buf).Decode(cronJob)
	if err != nil {
		return err
	}

	jobs := cs.BatchV1beta1().CronJobs("kube-system")
	_, err = jobs.Update(cronJob)
	return err
}

func (c updateEtcdBackupCronJobCommand) Command() cke.Command {
	return cke.Command{
		Name:   "update-etcdbackup-job",
		Target: "etcdbackup",
	}
}
