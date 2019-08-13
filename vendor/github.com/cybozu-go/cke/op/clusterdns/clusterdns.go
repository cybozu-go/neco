package clusterdns

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/cybozu-go/cke/op"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CoreDNSTemplateVersion is the version of CoreDNS template
const CoreDNSTemplateVersion = "2"

var clusterDNSTemplate = template.Must(template.New("").Parse(`.:1053 {
    errors
    health
    log
    kubernetes {{ .Domain }} in-addr.arpa ip6.arpa {
      pods verified
{{- if .Upstreams }}
      fallthrough in-addr.arpa ip6.arpa
{{- end }}
    }
{{- if .Upstreams }}
    forward . {{ .Upstreams }}
{{- end }}
    cache 30
    reload
    loadbalance
}
`))

// ConfigMap returns ConfigMap for CoreDNS
func ConfigMap(domain string, dnsServers []string) *corev1.ConfigMap {
	buf := new(bytes.Buffer)
	err := clusterDNSTemplate.Execute(buf, struct {
		Domain    string
		Upstreams string
	}{
		Domain:    domain,
		Upstreams: strings.Join(dnsServers, " "),
	})
	if err != nil {
		panic(err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      op.ClusterDNSAppName,
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"Corefile": buf.String(),
		},
	}
}
