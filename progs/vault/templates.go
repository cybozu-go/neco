package vault

import "text/template"

var confTmpl = template.Must(template.New("vault.hcl").
	Parse(`# vault configuration file

listener "tcp" {
    address = "0.0.0.0:8200"
    tls_cert_file = "{{ .ServerCertFile }}"
    tls_key_file = "{{ .ServerKeyFile }}"
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
Wants=network-online.target
After=network-online.target

[Service]
Slice=machine.slice
Type=simple
KillMode=mixed
Restart=always
RestartSec=10s
OOMScoreAdjust=-1000
LimitCORE=0
LimitMEMLOCK=infinity
ExecStart=/usr/bin/rkt run \
  --pull-policy never --net=host \
  --volume neco,kind=host,source=/etc/neco,readOnly=true \
  --mount volume=neco,target=/etc/neco \
  --volume certs,kind=host,source=/etc/ssl/certs,readOnly=true \
  --mount volume=certs,target=/etc/ssl/certs \
  --volume conf,kind=host,source=/etc/vault,readOnly=true \
  --mount volume=conf,target=/etc/vault \
  {{ .Image }} \
    --user {{ .UID }} --group {{ .GID }} \
    --name vault \
    --readonly-rootfs=true \
  -- \
    server -config={{ .ConfFile }}
ExecStartPost={{ .NecoBin }} vault unseal

[Install]
WantedBy=multi-user.target
`))
