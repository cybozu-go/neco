package op

import (
	"context"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
)

type kubeNodeUpdate struct {
	apiserver *cke.Node
	nodes     []*corev1.Node
	done      bool
}

// KubeNodeUpdateOp updates k8s Node resources.
func KubeNodeUpdateOp(apiserver *cke.Node, nodes []*corev1.Node) cke.Operator {
	return &kubeNodeUpdate{apiserver: apiserver, nodes: nodes}
}

func (o *kubeNodeUpdate) Name() string {
	return "update-node"
}

func (o *kubeNodeUpdate) NextCommand() cke.Commander {
	if o.done {
		return nil
	}

	o.done = true
	return nodeUpdateCommand{o.apiserver, o.nodes}
}

func (o *kubeNodeUpdate) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Name
	}
	return ips
}

type nodeUpdateCommand struct {
	apiserver *cke.Node
	nodes     []*corev1.Node
}

func (c nodeUpdateCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	nodesAPI := cs.CoreV1().Nodes()
	for _, n := range c.nodes {
		_, err := nodesAPI.Update(n)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c nodeUpdateCommand) Command() cke.Command {
	names := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		names[i] = n.Name
	}
	return cke.Command{
		Name: "updateNode",
	}
}
