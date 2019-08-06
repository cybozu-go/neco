package etcd

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type etcdStartOp struct {
	nodes  []*cke.Node
	params cke.EtcdParams
	step   int
	files  *common.FilesBuilder
	domain string
}

// StartOp returns an Operator to start etcd containers.
func StartOp(nodes []*cke.Node, params cke.EtcdParams, domain string) cke.Operator {
	return &etcdStartOp{
		nodes:  nodes,
		params: params,
		files:  common.NewFilesBuilder(nodes),
		domain: domain,
	}
}

func (o *etcdStartOp) Name() string {
	return "etcd-start"
}

func (o *etcdStartOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return prepareEtcdCertificatesCommand{o.files, o.domain}
	case 1:
		o.step++
		return o.files
	case 2:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + op.EtcdVolumeName(o.params) + ",dst=/var/lib/etcd",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = BuiltInParams(n, nil, "")
		}
		return common.RunContainerCommand(o.nodes, op.EtcdContainerName, cke.EtcdImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params.ServiceParams))
	case 3:
		o.step++
		return waitEtcdSyncCommand{etcdEndpoints(o.nodes), false}
	default:
		return nil
	}
}

func (o *etcdStartOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}
