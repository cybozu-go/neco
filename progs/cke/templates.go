package cke

import "html/template"

var ckePolicy = `
path "cke/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
`

var confTmpl = template.Must(template.New("config.yml").
	Parse(`endpoints: {{ .EtcdEndpoints }}
tls-cert-file: {{ .EtcdCertFile }}
tls-key-file: {{ .EtcdKeyFile }}
`))

var serviceTmpl = template.Must(template.New("cke.service").
	Parse(`[Unit]
Description=CKE container
Wants=network-online.target
After=network-online.target

[Service]
Slice=machine.slice
Type=simple
KillMode=mixed
Restart=always
RestartSec=10s
ExecStart=/usr/bin/rkt run \
  --pull-policy never \
  --net=host \
  --dns=host \
  --hosts-entry=host \
  --volume certs,kind=host,source=/etc/ssl/certs,readOnly=true \
  --mount volume=certs,target=/etc/ssl/certs \
  --volume conf,kind=host,source=/etc/cke,readOnly=true \
  --mount volume=conf,target=/etc/cke \
  {{ .Image }} \
    --name cke \
    --readonly-rootfs=true \
  -- \
    -config={{ .ConfFile }}

[Install]
WantedBy=multi-user.target
`))
