package sabakan

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/cybozu-go/neco"
)

// InstallTools installs sabactl and sabakan-cryptsetup under /usr/local/bin
func InstallTools(ctx context.Context) error {
	return neco.RunContainer(ctx, "sabakan",
		[]neco.Bind{
			{Name: "host-usr-local-bin", Source: "/usr/local/bin", Dest: "/host/usr/local/bin"},
		},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/sabakan/install-tools"})
}

// InstallBashCompletion installs bash completion for sabactl
func InstallBashCompletion(ctx context.Context) error {
	output, err := exec.Command(neco.SabactlBin, "completion").Output()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(neco.SabactlBashCompletionFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
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
