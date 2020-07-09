package ingresswatcher

import (
	"fmt"
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateConfBase generates ingress-watcher.yaml.base from template.
func GenerateConfBase(w io.Writer, lrn int) error {
	cluster, err := neco.MyCluster()
	if err != nil {
		return err
	}
	return confTmpl.Execute(w, struct {
		TargetURLs  []string
		PushAddress string
		LRN         int
	}{
		TargetURLs: []string{
			fmt.Sprintf("ingress-health-global.monitoring.%s.cybozu-ne.co", cluster),
			fmt.Sprintf("ingress-health-bastion.monitoring.%s.cybozu-ne.co", cluster),
		},
		PushAddress: fmt.Sprintf("pushgateway-bastion.monitoring.%s.cybozu-ne.co", cluster),
		LRN:         lrn,
	})

}
