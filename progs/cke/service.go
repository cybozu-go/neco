package cke

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateService generate systemd service unit contents.
func GenerateService(w io.Writer) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("cke")
	if err != nil {
		return err
	}
	return serviceTmpl.Execute(w, struct {
		Image    string
		ConfFile string
	}{
		Image:    img.FullName(),
		ConfFile: neco.CKEConfFile,
	})
}
