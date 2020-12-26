package vault

import (
	"context"

	"github.com/cybozu-go/neco"
)

// InstallTools install vault under /usr/local/bin.
func InstallTools(ctx context.Context, rt neco.ContainerRuntime) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("vault")
	if err != nil {
		return err
	}
	return rt.Run(ctx, img,
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"/usr/local/vault/install-tools"})
}
