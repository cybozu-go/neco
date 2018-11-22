package cke

import (
	"fmt"
	"io"

	"github.com/cybozu-go/neco"
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
