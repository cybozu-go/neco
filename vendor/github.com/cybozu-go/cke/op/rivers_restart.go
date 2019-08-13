package op

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op/common"
)

type riversRestartOp struct {
	nodes        []*cke.Node
	upstreams    []*cke.Node
	params       cke.ServiceParams
	name         string
	upstreamPort int
	listenPort   int

	pulled   bool
	finished bool
}

// RiversRestartOp returns an Operator to restart rivers.
func RiversRestartOp(nodes, upstreams []*cke.Node, params cke.ServiceParams, name string, upstreamPort, listenPort int) cke.Operator {
	return &riversRestartOp{
		nodes:        nodes,
		upstreams:    upstreams,
		params:       params,
		name:         name,
		upstreamPort: upstreamPort,
		listenPort:   listenPort,
	}
}

func (o *riversRestartOp) Name() string {
	return o.name + "-restart"
}

func (o *riversRestartOp) NextCommand() cke.Commander {
	if !o.pulled {
		o.pulled = true
		return common.ImagePullCommand(o.nodes, cke.ToolsImage)
	}

	if !o.finished {
		o.finished = true
		return common.RunContainerCommand(o.nodes, o.name, cke.ToolsImage,
			common.WithParams(RiversParams(o.upstreams, o.upstreamPort, o.listenPort)),
			common.WithExtra(o.params),
			common.WithRestart())
	}
	return nil
}

func (o *riversRestartOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}
