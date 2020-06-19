package ingresswatcher

import "text/template"

var confTmpl = template.Must(template.New("ingress-watcher-bastion.yaml").
	Parse(`# Ingress watcher configurations.
targetURLs:
- http://{{ .TargetURL }}
- https://{{ .TargetURL }}
watchInterval: 10s

pushAddr: {{ .PushAddress }}
jobName: ingress-watcher-bastion-{{ .LRN }}
pushInterval: 10s
`))

var serviceTmpl = template.Must(template.New("ingress-watcher-bastion.service").
	Parse(`[Unit]
Description=Ingress Watcher for Bastion Network
Wants=network-online.target
After=network-online.target
StartLimitIntervalSec=600s
ConditionPathExists={{ .ConfigFile }}

[Service]
Slice=system.slice
Type=simple
Restart=on-failure
RestartSec=30s
ExecStart=/usr/bin/rkt run \
  --pull-policy never \
  --net=host \
  --dns=host \
  --hosts-entry=host \
  --volume conf,kind=host,source=/etc/ingress-watcher-bastion,readOnly=true \
  --mount volume=conf,target=/etc/ingress-watcher-bastion \
  {{ .Image }} \
    --name ingress-watcher-bastion \
    --readonly-rootfs=true \
  -- \
	push \
    --config={{ .ConfigFile }}

[Install]
WantedBy=multi-user.target
`))
