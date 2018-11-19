package etcdpasswd

import (
	"fmt"
	"io"

	"github.com/cybozu-go/neco"
	yaml "gopkg.in/yaml.v2"
)

// GenerateConf generates etcdpasswd config file
func GenerateConf(w io.Writer, lrns []int) error {
	endpoints := make([]string, len(lrns))
	for i, lrn := range lrns {
		ip := neco.BootNode0IP(lrn).String()
		endpoints[i] = fmt.Sprintf("https://%s:2379", ip)
	}
	data := map[string]interface{}{
		"endpoints":     endpoints,
		"tls-cert-file": neco.EtcdpasswdCertFile,
		"tls-key-file":  neco.EtcdpasswdKeyFile,
	}
	return yaml.NewEncoder(w).Encode(data)
}
