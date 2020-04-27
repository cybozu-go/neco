package nodedns

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type createConfigMapOp struct {
	apiserver  *cke.Node
	clusterIP  string
	domain     string
	dnsServers []string
	finished   bool
}

// CreateConfigMapOp returns an Operator to create ConfigMap for unbound daemonset.
func CreateConfigMapOp(apiserver *cke.Node, clusterIP, domain string, dnsServers []string) cke.Operator {
	return &createConfigMapOp{
		apiserver:  apiserver,
		clusterIP:  clusterIP,
		domain:     domain,
		dnsServers: dnsServers,
	}
}

func (o *createConfigMapOp) Name() string {
	return "create-node-dns-configmap"
}

func (o *createConfigMapOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createConfigMapCommand{o.apiserver, o.clusterIP, o.domain, o.dnsServers}
}

func (o *createConfigMapOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type createConfigMapCommand struct {
	apiserver  *cke.Node
	clusterIP  string
	domain     string
	dnsServers []string
}

func (c createConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// ConfigMap
	configs := cs.CoreV1().ConfigMaps("kube-system")
	_, err = configs.Get(op.NodeDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		configMap := ConfigMap(c.clusterIP, c.domain, c.dnsServers)
		_, err = configs.Create(configMap)
		if err != nil {
			return err
		}
	default:
		return err
	}

	return nil
}

func (c createConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createConfigMapCommand",
		Target: "kube-system",
	}
}
