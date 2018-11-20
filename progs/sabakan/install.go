package sabakan

import (
	"context"

	"github.com/cybozu-go/neco"
)

// InstallTools installs sabactl and sabakan-cryptsetup under /usr/local/bin
func InstallTools(ctx context.Context) error {
	return neco.RunContainer(ctx, "sabakan",
		[]neco.Bind{
			{Name: "host-usr-local-bin", Source: "/usr/local/bin", Dest: "/host/usr/local/bin"},
			{Name: "host-etc-bash-completion-d", Source: "/etc/bash_completion.d", Dest: "/host/etc/bash_completion.d"},
		},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/sabakan/install-tools"})
}
