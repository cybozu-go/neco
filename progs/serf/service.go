package serf

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateService generate systemd service unit contents.
func GenerateService(w io.Writer, rt neco.ContainerRuntime) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("serf")
	if err != nil {
		return err
	}

	tmplArgs := struct {
		Image    string
		ConfFile string
	}{
		Image:    rt.ImageFullName(img),
		ConfFile: neco.SerfConfFile,
	}

	return serviceTmpl.Execute(w, tmplArgs)
}
