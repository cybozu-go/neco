package etcdbackup

import (
	"bytes"
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type etcdBackupCronJobCreateOp struct {
	apiserver *cke.Node
	schedule  string
	finished  bool
}

// CronJobCreateOp returns an Operator to create etcdbackup cron job.
func CronJobCreateOp(apiserver *cke.Node, schedule string) cke.Operator {
	return &etcdBackupCronJobCreateOp{
		apiserver: apiserver,
		schedule:  schedule,
	}
}

func (o *etcdBackupCronJobCreateOp) Name() string {
	return "etcdbackup-job-create"
}

func (o *etcdBackupCronJobCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createEtcdBackupCronJobCommand{o.apiserver, o.schedule}
}

func (o *etcdBackupCronJobCreateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type createEtcdBackupCronJobCommand struct {
	apiserver *cke.Node
	schedule  string
}

func (c createEtcdBackupCronJobCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	jobs := cs.BatchV1beta1().CronJobs("kube-system")
	_, err = jobs.Get(op.EtcdBackupAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		buf := new(bytes.Buffer)
		err := cronJobTemplate.Execute(buf, struct {
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
		_, err = jobs.Create(cronJob)
		if err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

func (c createEtcdBackupCronJobCommand) Command() cke.Command {
	return cke.Command{
		Name:   "create-etcdbackup-job",
		Target: "etcdbackup",
	}
}
