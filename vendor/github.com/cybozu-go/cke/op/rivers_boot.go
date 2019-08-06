package op

import (
	"fmt"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op/common"
)

type riversBootOp struct {
	nodes        []*cke.Node
	upstreams    []*cke.Node
	params       cke.ServiceParams
	step         int
	name         string
	upstreamPort int
	listenPort   int
}

// RiversBootOp returns an Operator to bootstrap rivers.
func RiversBootOp(nodes, upstreams []*cke.Node, params cke.ServiceParams, name string, upstreamPort, listenPort int) cke.Operator {
	return &riversBootOp{
		nodes:        nodes,
		upstreams:    upstreams,
		params:       params,
		name:         name,
		upstreamPort: upstreamPort,
		listenPort:   listenPort,
	}
}

func (o *riversBootOp) Name() string {
	return o.name + "-bootstrap"
}

func (o *riversBootOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.ToolsImage)
	case 1:
		o.step++
		return common.RunContainerCommand(o.nodes, o.name, cke.ToolsImage,
			common.WithParams(RiversParams(o.upstreams, o.upstreamPort, o.listenPort)),
			common.WithExtra(o.params))
	default:
		return nil
	}
}

// RiversParams returns parameters for rivers.
func RiversParams(upstreams []*cke.Node, upstreamPort, listenPort int) cke.ServiceParams {
	var ups []string
	for _, n := range upstreams {
		ups = append(ups, fmt.Sprintf("%s:%d", n.Address, upstreamPort))
	}
	args := []string{
		"rivers",
		"--upstreams=" + strings.Join(ups, ","),
		"--listen=" + fmt.Sprintf("127.0.0.1:%d", listenPort),
	}
	return cke.ServiceParams{ExtraArguments: args}
}

func (o *riversBootOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}
