package etcd

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateService generate systemd service unit contents.
func GenerateService(w io.Writer, rt neco.ContainerRuntime) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("etcd")
	if err != nil {
		return err
	}

	return serviceTmpl.Execute(w, struct {
		Image    string
		UID      int
		GID      int
		ConfFile string
	}{
		Image:    rt.ImageFullName(img),
		UID:      neco.EtcdUID,
		GID:      neco.EtcdGID,
		ConfFile: neco.EtcdConfFile,
	})
}
