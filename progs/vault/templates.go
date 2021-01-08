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
  prometheus_retention_time = "5m"
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
