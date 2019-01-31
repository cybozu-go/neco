package sabakan

import "text/template"

var serviceTmpl = template.Must(template.New("sabakan.service").Parse(`[Unit]
Description=Sabakan container on rkt
Wants=network-online.target
After=network-online.target
ConditionPathExists={{ .ConfFile }}
ConditionPathExists={{ .CertFile }}
ConditionPathExists={{ .KeyFile }}

[Service]
Slice=machine.slice
Type=simple
KillMode=mixed
Restart=always
RestartSec=10s
ExecStart=/usr/bin/rkt run \
  --pull-policy never --net=host \
  --volume neco,kind=host,source=/etc/neco,readOnly=true \
  --mount volume=neco,target=/etc/neco \
  --volume certs,kind=host,source=/etc/ssl/certs,readOnly=true \
  --mount volume=certs,target=/etc/ssl/certs \
  --volume conf,kind=host,source=/etc/sabakan,readOnly=true \
  --mount volume=conf,target=/etc/sabakan \
  --volume data,kind=host,source=/var/lib/sabakan \
  --mount volume=data,target=/var/lib/sabakan \
  {{ .Image }} \
    --name sabakan \
    --readonly-rootfs=true \
    --caps-retain=CAP_NET_BIND_SERVICE \
  -- \
  --config-file={{ .ConfFile }}

[Install]
WantedBy=multi-user.target
`))
