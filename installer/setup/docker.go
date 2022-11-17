package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/cybozu-go/neco"
)

var (
	dockerPrerequisites = []string{
		"apt-transport-https",
		"ca-certificates",
		"gnupg-agent",
		"software-properties-common",
	}

	dockerPackages = []string{
		"docker-ce",
		"docker-ce-cli",
		"containerd.io",
	}
)

func setupDocker() error {
	fmt.Fprintln(os.Stderr, "Installing docker...")

	s := fmt.Sprintf(`[Service]
Environment="HTTP_PROXY=%s"
Environment="HTTPS_PROXY=%s"
`, config.proxy, config.proxy)
	if err := installOverrideConf("docker", s); err != nil {
		return err
	}

	if err := installPackages(dockerPrerequisites...); err != nil {
		return err
	}

	resp, err := config.httpClient.Get("https://download.docker.com/linux/ubuntu/gpg")
	if err != nil {
		return fmt.Errorf("failed to retrieve Docker PGP key: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Docker PGP key: %w", err)
	}

	if err := os.WriteFile(neco.DockerKeyringFile, data, 0644); err != nil {
		return fmt.Errorf("failed to create %s: %w", neco.DockerKeyringFile, err)
	}

	cmd := exec.Command("lsb_release", "-cs")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to invoke lsb_release -cs: %w", err)
	}

	codename := strings.TrimSuffix(string(out), "\n")
	repo := fmt.Sprintf("deb [arch=amd64 signed-by=%s] https://download.docker.com/linux/ubuntu %s stable\n", neco.DockerKeyringFile, codename)
	if err := os.WriteFile(neco.DockerSourceListFile, []byte(repo), 0644); err != nil {
		return fmt.Errorf("failed to create %s: %w", neco.DockerSourceListFile, err)
	}

	if err := runCmd("apt-get", "update"); err != nil {
		return fmt.Errorf("failed to update packages: %w", err)
	}

	return installPackages(dockerPackages...)
}
