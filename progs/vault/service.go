package vault

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateService generate systemd service unit contents.
func GenerateService(w io.Writer, rt neco.ContainerRuntime) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("vault")
	if err != nil {
		return err
	}

	codename, err := neco.OSCodename()
	if err != nil {
		return err
	}

	tmplArgs := struct {
		Image    string
		UID      int
		GID      int
		ConfFile string
		NecoBin  string
	}{
		Image:    rt.ImageFullName(img),
		UID:      neco.VaultUID,
		GID:      neco.VaultGID,
		ConfFile: neco.VaultConfFile,
		NecoBin:  neco.NecoBin,
	}

	if codename == "bionic" {
		return serviceTmplRkt.Execute(w, tmplArgs)
	}
	return serviceTmpl.Execute(w, tmplArgs)
}
