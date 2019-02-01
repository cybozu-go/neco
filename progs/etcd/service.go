package etcd

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateService generate systemd service unit contents.
func GenerateService(w io.Writer) error {
	fullname, err := neco.ContainerFullName("etcd")
	if err != nil {
		return err
	}

	return serviceTmpl.Execute(w, struct {
		Image    string
		UID      int
		GID      int
		ConfFile string
	}{
		Image:    fullname,
		UID:      neco.EtcdUID,
		GID:      neco.EtcdGID,
		ConfFile: neco.EtcdConfFile,
	})
}
