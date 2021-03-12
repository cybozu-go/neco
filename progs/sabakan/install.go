package sabakan

import (
	"context"
	"os"
	"os/exec"

	"github.com/cybozu-go/neco"
)

// InstallTools installs sabactl and sabakan-cryptsetup under /usr/local/bin
func InstallTools(ctx context.Context, rt neco.ContainerRuntime) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("sabakan")
	if err != nil {
		return err
	}
	return rt.Run(ctx, img,
		[]neco.Bind{{Name: "host-usr-local-bin", Source: "/usr/local/bin", Dest: "/host/usr/local/bin"}},
		[]string{"/usr/local/sabakan/install-tools"})
}

// InstallBashCompletion installs bash completion for sabactl
func InstallBashCompletion(ctx context.Context) error {
	output, err := exec.Command(neco.SabactlBin, "completion").Output()
	if err != nil {
		return err
	}

	return os.WriteFile(neco.SabactlBashCompletionFile, output, 0644)
}
