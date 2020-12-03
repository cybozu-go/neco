package cke

import (
	"context"
	"fmt"
	"io"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const (
	ckeLabelRole   = "cke.cybozu.com/role"
	ckeLabelWeight = "cke.cybozu.com/weight"
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
func GenerateCKETemplate(ctx context.Context, st storage.Storage, name string, ckeTemplate []byte) ([]byte, error) {
	var tmpl map[string]interface{}
	err := yaml.Unmarshal(ckeTemplate, &tmpl)
	if err != nil {
		return nil, err
	}

	err = unstructured.SetNestedField(tmpl, name, "name")
	if err != nil {
		return nil, err
	}

	if name == "stage0" {
		if r, ok := tmpl["reboot"].(map[string]interface{}); ok {
			r["protected_namespaces"] = map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"team": "neco",
				},
			}
		}
	}

	weights, err := st.GetCKEWeight(ctx)
	if err != nil && err != storage.ErrNotFound {
		return nil, err
	}

	nodes, ok := tmpl["nodes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("nodes should be a slice")
	}
	for role, weight := range weights {
		for _, n := range nodes {
			n, ok := n.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("node should be a map")
			}
			cp, found, err := unstructured.NestedBool(n, "control_plane")
			if !found {
				return nil, fmt.Errorf("control_plane is not found")
			}
			if err != nil {
				return nil, err
			}
			labels, ok := n["labels"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("labels should be a map")
			}
			if !cp && labels[ckeLabelRole] == role {
				labels[ckeLabelWeight] = fmt.Sprintf("%f", weight)
				break
			}
		}
	}

	return yaml.Marshal(tmpl)
}
