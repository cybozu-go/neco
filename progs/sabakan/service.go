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

	tmplArgs := struct {
		Image    string
		ConfFile string
		CertFile string
		KeyFile  string
	}{
		Image:    rt.ImageFullName(img),
		ConfFile: neco.SabakanConfFile,
		CertFile: neco.SabakanEtcdCertFile,
		KeyFile:  neco.SabakanEtcdKeyFile,
	}

	return serviceTmpl.Execute(w, tmplArgs)
}
