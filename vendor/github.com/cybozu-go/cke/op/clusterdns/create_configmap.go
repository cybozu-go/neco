package clusterdns

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type createConfigMapOp struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
	finished   bool
}

// CreateConfigMapOp returns an Operator to create ConfigMap for CoreDNS.
func CreateConfigMapOp(apiserver *cke.Node, domain string, dnsServers []string) cke.Operator {
	return &createConfigMapOp{
		apiserver:  apiserver,
		domain:     domain,
		dnsServers: dnsServers,
	}
}

func (o *createConfigMapOp) Name() string {
	return "create-cluster-dns-configmap"
}

func (o *createConfigMapOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createConfigMapCommand{o.apiserver, o.domain, o.dnsServers}
}

func (o *createConfigMapOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

func (c createConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createConfigMapCommand",
		Target: "kube-system",
	}
}

type createConfigMapCommand struct {
	apiserver  *cke.Node
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
	_, err = configs.Get(op.ClusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = configs.Create(ConfigMap(c.domain, c.dnsServers))
		if err != nil {
			return err
		}
	default:
		return err
	}

	return nil
}
