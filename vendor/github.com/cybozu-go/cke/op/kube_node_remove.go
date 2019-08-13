package op

import (
	"context"
	"strings"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
)

type kubeNodeRemove struct {
	apiserver *cke.Node
	nodes     []*corev1.Node
	done      bool
}

// KubeNodeRemoveOp removes k8s Node resources.
func KubeNodeRemoveOp(apiserver *cke.Node, nodes []*corev1.Node) cke.Operator {
	return &kubeNodeRemove{apiserver: apiserver, nodes: nodes}
}

func (o *kubeNodeRemove) Name() string {
	return "remove-node"
}

func (o *kubeNodeRemove) NextCommand() cke.Commander {
	if o.done {
		return nil
	}

	o.done = true
	return nodeRemoveCommand{o.apiserver, o.nodes}
}

func (o *kubeNodeRemove) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type nodeRemoveCommand struct {
	apiserver *cke.Node
	nodes     []*corev1.Node
}

func (c nodeRemoveCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	nodesAPI := cs.CoreV1().Nodes()
	for _, n := range c.nodes {
		err := nodesAPI.Delete(n.Name, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c nodeRemoveCommand) Command() cke.Command {
	names := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		names[i] = n.Name
	}
	return cke.Command{
		Name:   "removeNode",
		Target: strings.Join(names, ","),
	}
}
