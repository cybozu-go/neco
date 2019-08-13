package etcdbackup

import (
	"bytes"
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type etcdBackupServiceCreateOp struct {
	apiserver *cke.Node
	finished  bool
}

// ServiceCreateOp returns an Operator to create etcdbackup service.
func ServiceCreateOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupServiceCreateOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupServiceCreateOp) Name() string {
	return "etcdbackup-service-create"
}

func (o *etcdBackupServiceCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createEtcdBackupServiceCommand{o.apiserver}
}

func (o *etcdBackupServiceCreateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type createEtcdBackupServiceCommand struct {
	apiserver *cke.Node
}

func (c createEtcdBackupServiceCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	services := cs.CoreV1().Services("kube-system")
	_, err = services.Get(op.EtcdBackupAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		Service := new(corev1.Service)
		err = k8sYaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(serviceText))).Decode(Service)
		if err != nil {
			return err
		}
		_, err = services.Create(Service)
		if err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

func (c createEtcdBackupServiceCommand) Command() cke.Command {
	return cke.Command{
		Name:   "create-etcdbackup-service",
		Target: "etcdbackup",
	}
}
