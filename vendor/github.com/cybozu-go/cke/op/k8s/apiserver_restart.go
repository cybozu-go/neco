package k8s

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type apiServerRestartOp struct {
	nodes []*cke.Node
	cps   []*cke.Node

	serviceSubnet string
	domain        string
	params        cke.APIServerParams

	step  int
	files *common.FilesBuilder
}

// APIServerRestartOp returns an Operator to restart kube-apiserver
func APIServerRestartOp(nodes, cps []*cke.Node, serviceSubnet, domain string, params cke.APIServerParams) cke.Operator {
	return &apiServerRestartOp{
		nodes:         nodes,
		cps:           cps,
		serviceSubnet: serviceSubnet,
		domain:        domain,
		params:        params,
		files:         common.NewFilesBuilder(nodes),
	}
}

func (o *apiServerRestartOp) Name() string {
	return "kube-apiserver-restart"
}

func (o *apiServerRestartOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.KubernetesImage)
	case 1:
		o.step++
		return prepareAPIServerFilesCommand{o.files, o.serviceSubnet, o.domain, o.params}
	case 2:
		o.step++
		return o.files
	case 3:
		if len(o.nodes) == 0 {
			return nil
		}

		// apiserver need to be restarted one by one
		node := o.nodes[0]
		o.nodes = o.nodes[1:]
		opts := []string{
			"--mount", "type=tmpfs,dst=/run/kubernetes",
		}
		return common.RunContainerCommand([]*cke.Node{node},
			op.KubeAPIServerContainerName, cke.KubernetesImage,
			common.WithOpts(opts),
			common.WithParams(APIServerParams(o.cps, node.Address, o.serviceSubnet, o.params.AuditLogEnabled, o.params.AuditLogPolicy)),
			common.WithExtra(o.params.ServiceParams),
			common.WithRestart())
	}

	panic("unreachable")
}

func (o *apiServerRestartOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}
