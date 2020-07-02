package ingresswatcher

import "text/template"

var confTmpl = template.Must(template.New("ingress-watcher.yaml").
	Parse(`# Ingress watcher configurations.
targetURLs:
{{- range .TargetURLs }}
- http://{{ . }}
- https://{{ . }}
{{- end }}
watchInterval: 10s

pushAddr: {{ .PushAddress }}
jobName: ingress-watcher-{{ .LRN }}
pushInterval: 10s
`))

var serviceTmpl = template.Must(template.New("ingress-watcher.service").
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
ExecStart=/usr/sbin/ingress-watcher --config={{ .ConfigFile }}

[Install]
WantedBy=multi-user.target
`))
