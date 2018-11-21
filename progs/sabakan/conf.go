package sabakan

import (
	"io"

	"github.com/cybozu-go/neco"
	yaml "gopkg.in/yaml.v2"
)

// GenerateConf generates sabakan config file
func GenerateConf(w io.Writer, mylrn int, lrns []int) error {
	myip := neco.BootNode0IP(mylrn)
	endpoints := make([]string, len(lrns))
	for i, lrn := range lrns {
		ip := neco.BootNode0IP(lrn)
		endpoints[i] = "https://" + ip.String() + ":2379"
	}
	data := map[string]interface{}{
		"advertise-url": "http://" + myip.String() + ":10080",
		"dhcp-bind":     "0.0.0.0:67",
		"etcd": map[string]interface{}{
			"endpoints":     endpoints,
			"tls-cert-file": neco.SabakanCertFile,
			"tls-key-file":  neco.SabakanKeyFile,
		},
	}
	return yaml.NewEncoder(w).Encode(data)
}
