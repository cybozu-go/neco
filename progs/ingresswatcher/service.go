package ingresswatcher

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateService generates systemd service unit contents.
func GenerateService(w io.Writer) error {
	fullname, err := neco.ContainerFullName("ingress-watcher")
	if err != nil {
		return err
	}

	return serviceTmpl.Execute(w, struct {
		ConfigFile string
		Image      string
	}{
		ConfigFile: neco.IngressWatcherConfFile,
		Image:      fullname,
	})
}
