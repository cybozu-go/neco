package omsa

import (
	"io"
	"os/exec"
	"strings"

	"github.com/cybozu-go/neco"
)

// GenerateService generates systemd service unit contents.
func GenerateService(w io.Writer) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("omsa")
	if err != nil {
		return err
	}

	output, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return err
	}
	kernelRelease := strings.TrimSpace(string(output))

	return serviceTmpl.Execute(w, struct {
		Image         string
		KernelRelease string
	}{
		Image:         img.FullName(),
		KernelRelease: kernelRelease,
	})
}
