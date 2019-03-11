package cke

import (
	"context"
	"io/ioutil"
	"os/exec"

	"github.com/cybozu-go/neco"
)

// InstallToolsCKE installs ckecli under /usr/local/bin.
func InstallToolsCKE(ctx context.Context) error {
	return neco.RunContainer(ctx, "cke",
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/cke/install-tools"})
}

// InstallBashCompletion installs bash completion for ckecli
func InstallBashCompletion(ctx context.Context) error {
	output, err := exec.Command(neco.CKECLIBin, "completion").Output()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(neco.CKECLIBashCompletionFile, output, 0644)
}
