package promtail

import (
	_ "embed"
	"io"
	"text/template"

	"github.com/cybozu-go/neco"
)

var (
	//go:embed promtail.service.tmpl
	templateText string
	serviceTmpl  = template.Must(template.New("promtail.service").Parse(templateText))
)

// GenerateService generate systemd service unit contents.
func GenerateService(w io.Writer, rt neco.ContainerRuntime) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("promtail")
	if err != nil {
		return err
	}

	tmplArgs := struct {
		Image    string
		ConfFile string
	}{
		Image:    rt.ImageFullName(img),
		ConfFile: neco.PromtailConfFile,
	}

	return serviceTmpl.Execute(w, tmplArgs)
}
