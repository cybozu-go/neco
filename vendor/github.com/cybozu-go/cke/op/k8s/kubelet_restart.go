package k8s

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
	"k8s.io/client-go/tools/clientcmd"
)

type kubeletRestartOp struct {
	nodes []*cke.Node

	cluster string
	params  cke.KubeletParams

	step  int
	files *common.FilesBuilder
}

// KubeletRestartOp returns an Operator to restart kubelet
func KubeletRestartOp(nodes []*cke.Node, cluster string, params cke.KubeletParams) cke.Operator {
	return &kubeletRestartOp{
		nodes:   nodes,
		cluster: cluster,
		params:  params,
		files:   common.NewFilesBuilder(nodes),
	}
}

func (o *kubeletRestartOp) Name() string {
	return "kubelet-restart"
}

func (o *kubeletRestartOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.KubernetesImage)
	case 1:
		o.step++
		return prepareKubeletConfigCommand{o.cluster, o.params, o.files}
	case 2:
		o.step++
		return o.files
	case 3:
		o.step++
		opts := []string{
			"--pid=host",
			"--privileged",
			"--tmpfs=/tmp",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = KubeletServiceParams(n, o.params)
		}
		return common.RunContainerCommand(o.nodes, op.KubeletContainerName, cke.KubernetesImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params.ServiceParams),
			common.WithRestart())
	case 4:
		o.step++
		return waitForKubeletReadyCommand{o.nodes}
	default:
		return nil
	}
}

func (o *kubeletRestartOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

type prepareKubeletConfigCommand struct {
	cluster string
	params  cke.KubeletParams
	files   *common.FilesBuilder
}

func (c prepareKubeletConfigCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	caPath := op.K8sPKIPath("ca.crt")
	tlsCertPath := op.K8sPKIPath("kubelet.crt")
	tlsKeyPath := op.K8sPKIPath("kubelet.key")

	cfg := newKubeletConfiguration(tlsCertPath, tlsKeyPath, caPath, c.params)
	g := func(ctx context.Context, n *cke.Node) ([]byte, error) {
		cfg := cfg
		cfg.ClusterDNS = []string{n.Address}
		return encodeToYAML(&cfg)
	}
	err := c.files.AddFile(ctx, kubeletConfigPath, g)
	if err != nil {
		return err
	}

	f := func(ctx context.Context, n *cke.Node) (cert, key []byte, err error) {
		c, k, e := cke.KubernetesCA{}.IssueForKubelet(ctx, inf, n)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err = c.files.AddKeyPair(ctx, op.K8sPKIPath("kubelet"), f)
	if err != nil {
		return err
	}

	g = func(ctx context.Context, n *cke.Node) ([]byte, error) {
		cfg := kubeletKubeconfig(c.cluster, n, caPath, tlsCertPath, tlsKeyPath)
		return clientcmd.Write(*cfg)
	}
	return c.files.AddFile(ctx, kubeconfigPath, g)
}

func (c prepareKubeletConfigCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-kubelet-config",
	}
}
