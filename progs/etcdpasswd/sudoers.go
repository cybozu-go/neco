package etcdpasswd

import "os"

const (
	sudoers = "%sudo   ALL=(ALL:ALL) NOPASSWD: ALL\n"
)

// InstallSudoers installs "/etc/sudoers.d/cybozu" file safely.
func InstallSudoers() error {
	const dest = "/etc/sudoers.d/cybozu"
	_, err := os.Stat(dest)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	f, err := os.OpenFile("/etc/sudoers.d/.cybozu", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0440)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(sudoers)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}

	return os.Rename(f.Name(), dest)
}
