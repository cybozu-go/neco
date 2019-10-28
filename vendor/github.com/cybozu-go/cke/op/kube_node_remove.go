package op

import (
	"context"
	"fmt"
	"strings"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
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
		if !n.DeletionTimestamp.IsZero() {
			continue
		}
		if !n.Spec.Unschedulable {
			oldData, err := json.Marshal(n)
			if err != nil {
				return err
			}
			n.Spec.Unschedulable = true
			newData, err := json.Marshal(n)
			if err != nil {
				return err
			}
			patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, n)
			if err != nil {
				return fmt.Errorf("failed to create patch for node %s: %v", n.Name, err)
			}
			_, err = nodesAPI.Patch(n.Name, types.StrategicMergePatchType, patchBytes)
			if err != nil {
				return fmt.Errorf("failed to patch node %s: %v", n.Name, err)
			}
		}
		err = nodesAPI.Delete(n.Name, nil)
		if err != nil {
			return fmt.Errorf("failed to delete node %s: %v", n.Name, err)
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
