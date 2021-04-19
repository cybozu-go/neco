package promtail

import (
	_ "embed"
	"io"
	"text/template"

	"github.com/cybozu-go/neco"
)

var (
	//go:embed promtail.yaml.tmpl
	promtailYaml string
	confTmpl     = template.Must(template.New("promtail.service").Parse(promtailYaml))
)

// GenerateConf generates promtail config file
func GenerateConf(w io.Writer, mylrn int) error {
	return confTmpl.Execute(w, struct {
		ServerIP string
	}{
		ServerIP: neco.BootNode0IP(mylrn).String(),
	})
}
