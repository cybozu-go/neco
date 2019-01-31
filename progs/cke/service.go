package cke

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateService generate systemd service unit contents.
func GenerateService(w io.Writer) error {
	fullname, err := neco.ContainerFullName("cke")
	if err != nil {
		return err
	}

	return serviceTmpl.Execute(w, struct {
		Image    string
		ConfFile string
		CertFile string
		KeyFile  string
	}{
		Image:    fullname,
		ConfFile: neco.CKEConfFile,
		CertFile: neco.CKECertFile,
		KeyFile:  neco.CKEKeyFile,
	})
}
