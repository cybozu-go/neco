package op

import (
	"context"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubeEtcdServiceCreateOp struct {
	apiserver *cke.Node
	finished  bool
}

// KubeEtcdServiceCreateOp returns an Operator to create Service resource for etcd.
func KubeEtcdServiceCreateOp(apiserver *cke.Node) cke.Operator {
	return &kubeEtcdServiceCreateOp{
		apiserver: apiserver,
	}
}

func (o *kubeEtcdServiceCreateOp) Name() string {
	return "create-etcd-service"
}

func (o *kubeEtcdServiceCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return createEtcdServiceCommand{o.apiserver}
}

func (o *kubeEtcdServiceCreateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type kubeEtcdServiceUpdateOp struct {
	apiserver *cke.Node
	finished  bool
}

// KubeEtcdServiceUpdateOp returns an Operator to update Service resource for etcd.
func KubeEtcdServiceUpdateOp(apiserver *cke.Node) cke.Operator {
	return &kubeEtcdServiceUpdateOp{
		apiserver: apiserver,
	}
}

func (o *kubeEtcdServiceUpdateOp) Name() string {
	return "update-etcd-service"
}

func (o *kubeEtcdServiceUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return updateEtcdServiceCommand{o.apiserver}
}

func (o *kubeEtcdServiceUpdateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type createEtcdServiceCommand struct {
	apiserver *cke.Node
}

func (c createEtcdServiceCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	_, err = cs.CoreV1().Services(metav1.NamespaceSystem).Create(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: EtcdServiceName,
		},
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{{Port: 2379}},
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
		},
	})
	return err
}

func (c createEtcdServiceCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createEtcdServiceCommand",
		Target: "",
	}
}

type updateEtcdServiceCommand struct {
	apiserver *cke.Node
}

func (c updateEtcdServiceCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	_, err = cs.CoreV1().Services(metav1.NamespaceSystem).Update(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: EtcdServiceName,
		},
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{{Port: 2379}},
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
		},
	})

	return err
}

func (c updateEtcdServiceCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateEtcdServiceCommand",
		Target: "",
	}
}
