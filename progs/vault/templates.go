package vault

import "text/template"

var confTmpl = template.Must(template.New("vault.hcl").
	Parse(`# vault configuration file

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_cert_file = "{{ .ServerCertFile }}"
  tls_key_file = "{{ .ServerKeyFile }}"
  telemetry {
    unauthenticated_metrics_access = true
  }
}

telemetry {
  # Vault retains Prometheus-compatible metrics only for the following periods if the metrics do not change.
  # Some metrics do not change for several tens of minutes or more and such metrics disappear after the period.
  # We set VERY long period (about three years, enough longer than boot server OS upgrade interval) to keep the metrics.
  prometheus_retention_time = "1000d"
  disable_hostname = true
}

api_addr = "{{ .APIAddr }}"
cluster_addr = "{{ .ClusterAddr }}"

storage "etcd" {
  address = "{{ .EtcdEndpoints }}"
  etcd_api = "v3"
  ha_enabled = "true"
  tls_cert_file = "{{ .EtcdCertFile }}"
  tls_key_file = "{{ .EtcdKeyFile }}"
}
`))

var serviceTmpl = template.Must(template.New("vault.service").
	Parse(`[Unit]
Description=Vault container
Wants=network-online.target docker.service etcd-container.service
After=network-online.target docker.service etcd-container.service
StartLimitIntervalSec=600s

[Service]
Type=simple
Restart=always
RestartSec=10s
OOMScoreAdjust=-1000
ExecStartPre=-/usr/bin/docker kill vault
ExecStartPre=-/usr/bin/docker rm vault
ExecStart=/usr/bin/docker run --name=vault --rm \
  --network=host --uts=host \
  --log-driver=journald \
  --pull=never \
  --oom-kill-disable=true \
  --user={{ .UID }}:{{ .GID }} \
  --ulimit core=0 \
  --ulimit memlock=-1 \
  --read-only \
  --volume=/etc/neco:/etc/neco:ro \
  --volume=/etc/ssl/certs:/etc/ssl/certs:ro \
  --volume=/etc/vault:/etc/vault:ro \
  {{ .Image }} server -config={{ .ConfFile }}
ExecStartPost={{ .NecoBin }} vault unseal

[Install]
WantedBy=multi-user.target
`))
