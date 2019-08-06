package cke

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/sabakan"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	yaml "gopkg.in/yaml.v2"
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
	return yaml.NewEncoder(w).Encode(data)
}

// GenerateCKETemplate generates cke-template.yml from template.
func GenerateCKETemplate(ctx context.Context, w io.Writer, st storage.Storage) error {
	r, err := os.Open(neco.CKETemplateFile)
	if err != nil {
		return err
	}
	defer r.Close()

	tmpl := cke.NewCluster()
	err = yaml.NewDecoder(r).Decode(tmpl)
	if err != nil {
		return err
	}

	weights, err := st.GetCKEWeight(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}

	for role, weight := range weights {
		for _, n := range tmpl.Nodes {
			if n.Labels[sabakan.CKELabelRole] == role {
				n.Labels[sabakan.CKELabelWeight] = fmt.Sprintf("%f", weight)
				break
			}
		}
	}

	return yaml.NewEncoder(w).Encode(tmpl)
}
