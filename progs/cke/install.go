package cke

import (
	"bytes"
	"context"
	"io"
	"os"
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

	f, err := os.OpenFile(neco.CKECLIBashCompletionFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	bc := bytes.NewReader(output)
	_, err = io.Copy(f, bc)
	if err != nil {
		return err
	}

	return f.Sync()
}
