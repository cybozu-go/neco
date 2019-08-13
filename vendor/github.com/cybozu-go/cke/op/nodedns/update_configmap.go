package nodedns

import (
	"context"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
)

type updateConfigMapOp struct {
	apiserver *cke.Node
	configMap *corev1.ConfigMap
	finished  bool
}

// UpdateConfigMapOp returns an Operator to update unbound as Node local resolver.
func UpdateConfigMapOp(apiserver *cke.Node, configMap *corev1.ConfigMap) cke.Operator {
	return &updateConfigMapOp{
		apiserver: apiserver,
		configMap: configMap,
	}
}

func (o *updateConfigMapOp) Name() string {
	return "update-node-dns-configmap"
}

func (o *updateConfigMapOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateConfigMapCommand{o.apiserver, o.configMap}
}

func (o *updateConfigMapOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type updateConfigMapCommand struct {
	apiserver *cke.Node
	configMap *corev1.ConfigMap
}

func (c updateConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	configs := cs.CoreV1().ConfigMaps("kube-system")
	_, err = configs.Update(c.configMap)
	return err
}

func (c updateConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateConfigMapCommand",
		Target: "kube-system",
	}
}
