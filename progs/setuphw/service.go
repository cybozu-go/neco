package setuphw

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateService generates systemd service unit contents.
func GenerateService(w io.Writer, rt neco.ContainerRuntime) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("setup-hw")
	if err != nil {
		return err
	}

	codename, err := neco.OSCodename()
	if err != nil {
		return err
	}

	tmplArgs := struct {
		Image string
	}{
		Image: rt.ImageFullName(img),
	}

	if codename == "bionic" {
		return serviceTmplRkt.Execute(w, tmplArgs)
	}
	return serviceTmpl.Execute(w, tmplArgs)
}
