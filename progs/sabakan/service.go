package sabakan

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateService generate systemd service unit contents.
func GenerateService(w io.Writer, rt neco.ContainerRuntime) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("sabakan")
	if err != nil {
		return err
	}

	return serviceTmpl.Execute(w, struct {
		Image    string
		ConfFile string
		CertFile string
		KeyFile  string
	}{
		Image:    rt.ImageFullName(img),
		ConfFile: neco.SabakanConfFile,
		CertFile: neco.SabakanCertFile,
		KeyFile:  neco.SabakanKeyFile,
	})
}
