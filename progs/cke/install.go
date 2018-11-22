package cke

import (
	"context"

	"github.com/cybozu-go/neco"
)

// InstallToolsCKE installs ckecli under /usr/local/bin.
func InstallToolsCKE(ctx context.Context) error {
	return neco.RunContainer(ctx, "cke",
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/cke/install-tools"})
}

// InstallToolsHyperKube installs kubectl under /usr/local/bin.
func InstallToolsHyperKube(ctx context.Context) error {
	return neco.RunContainer(ctx, "hyperkube",
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/kubernetes/install-tools"})
}
