package cke

import (
	"context"
	"fmt"
	"io"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/sabakan"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"sigs.k8s.io/yaml"
)

// GenerateConf generates config.yml from template.
func GenerateConf(w io.Writer, lrns []int) error {
	endpoints := make([]string, len(lrns))
	for i, lrn := range lrns {
		ip := neco.BootNode0IP(lrn).String()
		endpoints[i] = fmt.Sprintf("https://%s:2379", ip)
	}
	data := map[string]interface{}{
		"endpoints":     endpoints,
		"tls-cert-file": neco.CKECertFile,
		"tls-key-file":  neco.CKEKeyFile,
	}
	conf, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	_, err = w.Write(conf)
	return err
}

// GenerateCKETemplate generates cke-template.yml using role and weights.
func GenerateCKETemplate(ctx context.Context, st storage.Storage, ckeTemplate []byte) ([]byte, error) {
	tmpl := cke.NewCluster()
	err := yaml.Unmarshal(ckeTemplate, tmpl)
	if err != nil {
		return nil, err
	}

	weights, err := st.GetCKEWeight(ctx)
	if err != nil && err != storage.ErrNotFound {
		return nil, err
	}

	for role, weight := range weights {
		for _, n := range tmpl.Nodes {
			if !n.ControlPlane && n.Labels[sabakan.CKELabelRole] == role {
				n.Labels[sabakan.CKELabelWeight] = fmt.Sprintf("%f", weight)
				break
			}
		}
	}

	return yaml.Marshal(tmpl)
}
