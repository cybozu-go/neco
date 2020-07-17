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

instance: {{ .Instance }}
pushAddr: {{ .PushAddress }}
pushInterval: 10s
`))
