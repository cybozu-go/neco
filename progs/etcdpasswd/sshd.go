package etcdpasswd

import (
	"fmt"
	"os"
)

const (
	sshdConfFile = "/etc/ssh/sshd_config.d/neco.conf"
	sshdConf     = `# SSHD configurations for Neco

AuthorizedKeysFile	.ssh/authorized_keys
PasswordAuthentication no
ForceCommand /usr/bin/neco session-log start
`
)

// InstallSshdConf installs sshd_config file for Neco
func InstallSshdConf() error {
	tmpFile := sshdConfFile + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", tmpFile, err)
	}
	defer f.Close()

	if _, err := f.Write([]byte(sshdConf)); err != nil {
		return fmt.Errorf("failed to write to %s: %w", tmpFile, err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to fsync %s: %w", tmpFile, err)
	}
	if err := os.Rename(tmpFile, sshdConfFile); err != nil {
		return fmt.Errorf("failed to rename %s: %w", tmpFile, err)
	}
	return nil
}
