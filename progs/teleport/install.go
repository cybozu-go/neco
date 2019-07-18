package teleport

import (
	"context"

	"github.com/cybozu-go/neco"
)

// InstallTools install teleport under /usr/local/bin.
func InstallTools(ctx context.Context) error {
	return neco.RunContainer(ctx, "teleport",
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host/usr/local/bin"}},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/teleport/install-tools"})
}
