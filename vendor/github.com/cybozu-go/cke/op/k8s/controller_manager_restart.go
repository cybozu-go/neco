package k8s

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type controllerManagerRestartOp struct {
	nodes []*cke.Node

	cluster       string
	serviceSubnet string
	params        cke.ServiceParams

	pulled   bool
	finished bool
}

// ControllerManagerRestartOp returns an Operator to restart kube-controller-manager
func ControllerManagerRestartOp(nodes []*cke.Node, cluster, serviceSubnet string, params cke.ServiceParams) cke.Operator {
	return &controllerManagerRestartOp{
		nodes:         nodes,
		cluster:       cluster,
		serviceSubnet: serviceSubnet,
		params:        params,
	}
}

func (o *controllerManagerRestartOp) Name() string {
	return "kube-controller-manager-restart"
}

func (o *controllerManagerRestartOp) NextCommand() cke.Commander {
	if !o.pulled {
		o.pulled = true
		return common.ImagePullCommand(o.nodes, cke.HyperkubeImage)
	}

	if !o.finished {
		o.finished = true
		return common.RunContainerCommand(o.nodes, op.KubeControllerManagerContainerName, cke.HyperkubeImage,
			common.WithParams(ControllerManagerParams(o.cluster, o.serviceSubnet)),
			common.WithExtra(o.params),
			common.WithRestart())
	}
	return nil
}

func (o *controllerManagerRestartOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}
