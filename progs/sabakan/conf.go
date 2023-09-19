package sabakan

import (
	"io"

	"github.com/cybozu-go/neco"
	"sigs.k8s.io/yaml"
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
		"advertise-url":       "http://" + myip.String() + ":10080",
		"advertise-url-https": "https://" + myip.String() + ":10443",

		"dhcp-bind": "0.0.0.0:67",
		"etcd": map[string]interface{}{
			"endpoints":     endpoints,
			"tls-cert-file": neco.SabakanEtcdCertFile,
			"tls-key-file":  neco.SabakanEtcdKeyFile,
		},
	}
	b, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
