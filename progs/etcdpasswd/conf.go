package etcdpasswd

import (
	"fmt"
	"io"

	"github.com/cybozu-go/neco"
	"sigs.k8s.io/yaml"
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
	b, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// GenerateSystemdDropIn generates systemd drop-in file
func GenerateSystemdDropIn(w io.Writer) error {
	tmpl := `
[Unit]
Wants=etcd-container.service
After=etcd-container.service
ConditionPathExists=%s
ConditionPathExists=%s
StartLimitIntervalSec=600s

[Service]
RestartSec=30s
`
	_, err := fmt.Fprintf(w, tmpl, neco.EtcdpasswdCertFile, neco.EtcdpasswdKeyFile)
	return err
}
