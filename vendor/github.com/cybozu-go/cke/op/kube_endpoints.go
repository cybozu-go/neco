package op

import (
	"context"
	"strings"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
)

type kubeEndpointsCreateOp struct {
	apiserver *cke.Node
	endpoints *corev1.Endpoints
	finished  bool
}

// KubeEndpointsCreateOp returns an Operator to create Endpoints resource.
func KubeEndpointsCreateOp(apiserver *cke.Node, ep *corev1.Endpoints) cke.Operator {
	return &kubeEndpointsCreateOp{
		apiserver: apiserver,
		endpoints: ep,
	}
}

func (o *kubeEndpointsCreateOp) Name() string {
	return "create-endpoints"
}

func (o *kubeEndpointsCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return createEndpointsCommand{o.apiserver, o.endpoints}
}

func (o *kubeEndpointsCreateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type kubeEndpointsUpdateOp struct {
	apiserver *cke.Node
	endpoints *corev1.Endpoints
	finished  bool
}

// KubeEndpointsUpdateOp returns an Operator to update Endpoints resource.
func KubeEndpointsUpdateOp(apiserver *cke.Node, ep *corev1.Endpoints) cke.Operator {
	return &kubeEndpointsUpdateOp{
		apiserver: apiserver,
		endpoints: ep,
	}
}

func (o *kubeEndpointsUpdateOp) Name() string {
	return "update-endpoints"
}

func (o *kubeEndpointsUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return updateEndpointsCommand{o.apiserver, o.endpoints}
}

func (o *kubeEndpointsUpdateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type createEndpointsCommand struct {
	apiserver *cke.Node
	endpoints *corev1.Endpoints
}

func (c createEndpointsCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	_, err = cs.CoreV1().Endpoints(c.endpoints.Namespace).Create(c.endpoints)

	return err
}

func (c createEndpointsCommand) Command() cke.Command {
	endpoints := make([]string, len(c.endpoints.Subsets[0].Addresses))
	for i, e := range c.endpoints.Subsets[0].Addresses {
		endpoints[i] = e.IP
	}
	return cke.Command{
		Name:   "createEndpointsCommand",
		Target: strings.Join(endpoints, ","),
	}
}

type updateEndpointsCommand struct {
	apiserver *cke.Node
	endpoints *corev1.Endpoints
}

func (c updateEndpointsCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	_, err = cs.CoreV1().Endpoints(c.endpoints.Namespace).Update(c.endpoints)

	return err
}

func (c updateEndpointsCommand) Command() cke.Command {
	endpoints := make([]string, len(c.endpoints.Subsets[0].Addresses))
	for i, e := range c.endpoints.Subsets[0].Addresses {
		endpoints[i] = e.IP
	}
	return cke.Command{
		Name:   "updateEndpointsCommand",
		Target: strings.Join(endpoints, ","),
	}
}
