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

	return serviceTmpl.Execute(w, struct {
		Image string
	}{
		Image: rt.ImageFullName(img),
	})
}
