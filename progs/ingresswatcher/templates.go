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
