package serf

import (
	"context"

	"github.com/cybozu-go/neco"
)

// InstallTools install serf under /usr/local/bin.
func InstallTools(ctx context.Context, rt neco.ContainerRuntime) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("serf")
	if err != nil {
		return err
	}
	return rt.Run(ctx, img,
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host/usr/local/bin"}},
		[]string{"/usr/local/serf/install-tools"})
}
