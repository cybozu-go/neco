package serf

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateService generate systemd service unit contents.
func GenerateService(w io.Writer) error {
	fullname, err := neco.ContainerFullName("serf")
	if err != nil {
		return err
	}

	return serviceTmpl.Execute(w, struct {
		Image    string
		ConfFile string
	}{
		Image:    fullname,
		ConfFile: neco.SerfConfFile,
	})
}
