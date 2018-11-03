package vault

import (
	"context"

	"github.com/cybozu-go/neco"
)

// InstallTools install vault under /usr/local/bin.
func InstallTools(ctx context.Context) error {
	return neco.RunContainer(ctx, "vault",
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/vault/install-tools"})
}
