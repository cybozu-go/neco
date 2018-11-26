package omsa

import (
	"context"

	"github.com/cybozu-go/neco"
)

// InstallTools installs config file generated in OMSA container.
func InstallTools(ctx context.Context) error {
	return neco.RunContainer(ctx, "omsa",
		[]neco.Bind{
			{Name: "setup", Source: "/extras/setup", Dest: "/extras/setup"},
			{Name: "neco", Source: "/etc/neco", Dest: "/etc/neco"},
		},
		[]string{"--user=0", "--group=0", "--exec=install-tools"})
}
