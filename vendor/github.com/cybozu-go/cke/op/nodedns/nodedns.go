package nodedns

import (
	"bytes"
	"text/template"

	"github.com/cybozu-go/cke/op"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type unboundConfigTemplate struct {
	Domain    string
	ClusterIP string
	Upstreams []string
}

const unboundConfigTemplateText = `
server:
  do-daemonize: no
  interface: 0.0.0.0
  interface-automatic: yes
  access-control: 0.0.0.0/0 allow
  chroot: ""
  username: ""
  directory: "/etc/unbound"
  logfile: ""
  use-syslog: no
  log-time-ascii: yes
  log-queries: yes
  log-replies: yes
  log-local-actions: yes
  log-servfail: yes
  rrset-roundrobin: yes
  pidfile: "/tmp/unbound.pid"
  infra-host-ttl: 60
  prefetch: yes
  tcp-upstream: yes
  local-zone: "10.in-addr.arpa." transparent
  local-zone: "168.192.in-addr.arpa." transparent
  local-zone: "16.172.in-addr.arpa." transparent
  local-zone: "17.172.in-addr.arpa." transparent
  local-zone: "18.172.in-addr.arpa." transparent
  local-zone: "19.172.in-addr.arpa." transparent
  local-zone: "20.172.in-addr.arpa." transparent
  local-zone: "21.172.in-addr.arpa." transparent
  local-zone: "22.172.in-addr.arpa." transparent
  local-zone: "23.172.in-addr.arpa." transparent
  local-zone: "24.172.in-addr.arpa." transparent
  local-zone: "25.172.in-addr.arpa." transparent
  local-zone: "26.172.in-addr.arpa." transparent
  local-zone: "27.172.in-addr.arpa." transparent
  local-zone: "28.172.in-addr.arpa." transparent
  local-zone: "29.172.in-addr.arpa." transparent
  local-zone: "30.172.in-addr.arpa." transparent
  local-zone: "31.172.in-addr.arpa." transparent
remote-control:
  control-enable: yes
  control-interface: 127.0.0.1
  control-use-cert: no
stub-zone:
  name: "{{ .Domain }}"
  stub-addr: {{ .ClusterIP }}
forward-zone:
  name: "in-addr.arpa."
  forward-addr: {{ .ClusterIP }}
forward-zone:
  name: "ip6.arpa."
  forward-addr: {{ .ClusterIP }}
{{- if .Upstreams }}
forward-zone:
  name: "."
  {{- range .Upstreams }}
  forward-addr: {{ . }}
  {{- end }}
{{- end }}
`

// ConfigMap returns ConfigMap for unbound daemonset
func ConfigMap(clusterIP, domain string, dnsServers []string) *corev1.ConfigMap {
	var confTempl unboundConfigTemplate
	confTempl.Domain = domain
	confTempl.ClusterIP = clusterIP
	confTempl.Upstreams = dnsServers

	tmpl := template.Must(template.New("").Parse(unboundConfigTemplateText))
	unboundConf := new(bytes.Buffer)
	err := tmpl.Execute(unboundConf, confTempl)
	if err != nil {
		panic(err)
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      op.NodeDNSAppName,
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"unbound.conf": unboundConf.String(),
		},
	}
}
