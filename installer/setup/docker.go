package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Docker PGP key: %w", err)
	}

	cmd := exec.Command("apt-key", "add", "-")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add apt-key: %s: %w", string(out), err)
	}

	cmd = exec.Command("lsb_release", "-cs")
	codename, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to invoke lsb_release -cs: %w", err)
	}

	repo := fmt.Sprintf("deb [arch=amd64] https://download.docker.com/linux/ubuntu %s stable", string(codename))
	if err := runCmd("add-apt-repository", repo); err != nil {
		return err
	}

	return installPackages(dockerPackages...)
}
