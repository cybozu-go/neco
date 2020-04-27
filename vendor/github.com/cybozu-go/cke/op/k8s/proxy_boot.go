package k8s

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
	"k8s.io/client-go/tools/clientcmd"
)

type kubeProxyBootOp struct {
	nodes []*cke.Node

	cluster string
	params  cke.ServiceParams

	step  int
	files *common.FilesBuilder
}

// KubeProxyBootOp returns an Operator to boot kube-proxy.
func KubeProxyBootOp(nodes []*cke.Node, cluster string, params cke.ServiceParams) cke.Operator {
	return &kubeProxyBootOp{
		nodes:   nodes,
		cluster: cluster,
		params:  params,
		files:   common.NewFilesBuilder(nodes),
	}
}

func (o *kubeProxyBootOp) Name() string {
	return "kube-proxy-bootstrap"
}

func (o *kubeProxyBootOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.KubernetesImage)
	case 1:
		o.step++
		return prepareProxyFilesCommand{o.cluster, o.files}
	case 2:
		o.step++
		return o.files
	case 3:
		o.step++
		opts := []string{
			"--tmpfs=/run",
			"--privileged",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			params := ProxyParams(n)
			paramsMap[n.Address] = params
		}
		return common.RunContainerCommand(o.nodes, op.KubeProxyContainerName, cke.KubernetesImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params))
	default:
		return nil
	}
}

func (o *kubeProxyBootOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

type prepareProxyFilesCommand struct {
	cluster string
	files   *common.FilesBuilder
}

func (c prepareProxyFilesCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	const kubeconfigPath = "/etc/kubernetes/proxy/kubeconfig"
	storage := inf.Storage()

	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	g := func(ctx context.Context, n *cke.Node) ([]byte, error) {
		crt, key, err := cke.KubernetesCA{}.IssueForProxy(ctx, inf)
		if err != nil {
			return nil, err
		}
		cfg := proxyKubeconfig(c.cluster, ca, crt, key)
		return clientcmd.Write(*cfg)
	}
	return c.files.AddFile(ctx, kubeconfigPath, g)
}

func (c prepareProxyFilesCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-proxy-files",
	}
}

// ProxyParams returns parameters for kube-proxy.
func ProxyParams(n *cke.Node) cke.ServiceParams {
	args := []string{
		"kube-proxy",
		"--proxy-mode=ipvs",
		"--hostname-override=" + n.Nodename(),
		"--kubeconfig=/etc/kubernetes/proxy/kubeconfig",
	}
	return cke.ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []cke.Mount{
			{
				Source:      "/etc/machine-id",
				Destination: "/etc/machine-id",
				ReadOnly:    true,
				Propagation: "",
				Label:       "",
			},
			{
				Source:      "/etc/kubernetes",
				Destination: "/etc/kubernetes",
				ReadOnly:    true,
				Propagation: "",
				Label:       cke.LabelShared,
			},
			{
				Source:      "/lib/modules",
				Destination: "/lib/modules",
				ReadOnly:    true,
				Propagation: "",
				Label:       "",
			},
		},
	}
}
