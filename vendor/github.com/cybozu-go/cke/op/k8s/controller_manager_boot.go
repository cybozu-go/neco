package k8s

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
	"k8s.io/client-go/tools/clientcmd"
)

type controllerManagerBootOp struct {
	nodes []*cke.Node

	cluster       string
	serviceSubnet string
	params        cke.ServiceParams

	step  int
	files *common.FilesBuilder
}

// ControllerManagerBootOp returns an Operator to bootstrap kube-controller-manager
func ControllerManagerBootOp(nodes []*cke.Node, cluster string, serviceSubnet string, params cke.ServiceParams) cke.Operator {
	return &controllerManagerBootOp{
		nodes:         nodes,
		cluster:       cluster,
		serviceSubnet: serviceSubnet,
		params:        params,
		files:         common.NewFilesBuilder(nodes),
	}
}

func (o *controllerManagerBootOp) Name() string {
	return "kube-controller-manager-bootstrap"
}

func (o *controllerManagerBootOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.HyperkubeImage)
	case 1:
		o.step++
		return prepareControllerManagerFilesCommand{o.cluster, o.files}
	case 2:
		o.step++
		return o.files
	case 3:
		o.step++
		return common.RunContainerCommand(o.nodes,
			op.KubeControllerManagerContainerName, cke.HyperkubeImage,
			common.WithParams(ControllerManagerParams(o.cluster, o.serviceSubnet)),
			common.WithExtra(o.params))
	default:
		return nil
	}
}

func (o *controllerManagerBootOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

type prepareControllerManagerFilesCommand struct {
	cluster string
	files   *common.FilesBuilder
}

func (c prepareControllerManagerFilesCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	const kubeconfigPath = "/etc/kubernetes/controller-manager/kubeconfig"
	storage := inf.Storage()

	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	g := func(ctx context.Context, n *cke.Node) ([]byte, error) {
		crt, key, err := cke.KubernetesCA{}.IssueForControllerManager(ctx, inf)
		if err != nil {
			return nil, err
		}
		cfg := controllerManagerKubeconfig(c.cluster, ca, crt, key)
		return clientcmd.Write(*cfg)
	}
	err = c.files.AddFile(ctx, kubeconfigPath, g)
	if err != nil {
		return err
	}

	saKey, err := storage.GetServiceAccountKey(ctx)
	if err != nil {
		return err
	}
	saKeyData := []byte(saKey)
	g = func(ctx context.Context, n *cke.Node) ([]byte, error) {
		return saKeyData, nil
	}
	return c.files.AddFile(ctx, op.K8sPKIPath("service-account.key"), g)
}

func (c prepareControllerManagerFilesCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-controller-manager-files",
	}
}

// ControllerManagerParams returns parameters for kube-controller-manager.
func ControllerManagerParams(clusterName, serviceSubnet string) cke.ServiceParams {
	args := []string{
		"controller-manager",
		"--cluster-name=" + clusterName,
		"--service-cluster-ip-range=" + serviceSubnet,
		"--kubeconfig=/etc/kubernetes/controller-manager/kubeconfig",

		// ToDo: cluster signing
		// https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/#a-note-to-cluster-administrators
		// https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet-tls-bootstrapping/
		//    Create an intermediate CA under cke/ca-kubernetes?
		//    or just an certificate/key pair?
		// "--cluster-signing-cert-file=..."
		// "--cluster-signing-key-file=..."

		// for healthz service
		"--tls-cert-file=" + op.K8sPKIPath("apiserver.crt"),
		"--tls-private-key-file=" + op.K8sPKIPath("apiserver.key"),

		// for service accounts
		"--root-ca-file=" + op.K8sPKIPath("ca.crt"),
		"--service-account-private-key-file=" + op.K8sPKIPath("service-account.key"),
		"--use-service-account-credentials=true",
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
		},
	}
}
