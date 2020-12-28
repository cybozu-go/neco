package sabakan

import "text/template"

var serviceTmpl = template.Must(template.New("sabakan.service").Parse(`[Unit]
Description=Sabakan container
Wants=network-online.target docker.service etcd-container.service
After=network-online.target docker.service etcd-container.service
ConditionPathExists={{ .ConfFile }}
ConditionPathExists={{ .CertFile }}
ConditionPathExists={{ .KeyFile }}
StartLimitIntervalSec=600s

[Service]
Type=simple
Restart=always
RestartSec=30s
ExecStartPre=-/usr/bin/docker kill sabakan
ExecStartPre=-/usr/bin/docker rm sabakan
ExecStart=/usr/bin/docker run --name=sabakan --rm \
  --network=host --uts=host \
  --log-driver=journald \
  --pull=never \
  --read-only \
  --cap-add=NET_BIND_SERVICE \
  --volume=/etc/neco:/etc/neco:ro \
  --volume=/etc/ssl/certs:/etc/ssl/certs:ro \
  --volume=/etc/sabakan:/etc/sabakan:ro \
  --volume=/var/lib/sabakan:/var/lib/sabakan \
  {{ .Image }} --config-file={{ .ConfFile }}

[Install]
WantedBy=multi-user.target
`))

var serviceTmplRkt = template.Must(template.New("sabakan.service").Parse(`[Unit]
Description=Sabakan container on rkt
Wants=network-online.target etcd-container.service
After=network-online.target etcd-container.service
ConditionPathExists={{ .ConfFile }}
ConditionPathExists={{ .CertFile }}
ConditionPathExists={{ .KeyFile }}
StartLimitIntervalSec=600s

[Service]
Slice=machine.slice
Type=simple
KillMode=mixed
Restart=on-failure
RestartSec=30s
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
