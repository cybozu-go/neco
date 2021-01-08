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
