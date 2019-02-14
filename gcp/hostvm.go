package gcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
)

const (
	homeDisk           = "/dev/disk/by-id/google-home"
	homeFSType         = "ext4"
	homeMountPoint     = "/home"
	localSSDDisk       = "/dev/disk/by-id/google-local-ssd-0"
	localSSDFSType     = "ext4"
	localSSDMountPoint = "/var/scratch"
)

// SetupHostVM setup vmx-enabled instance
func SetupHostVM(ctx context.Context) error {
	err := enableXForwarding()
	if err != nil {
		return err
	}

	err = mountHomeDisk(ctx)
	if err != nil {
		return err
	}

	return setupLocalSSD(ctx)
}

func enableXForwarding() error {
	reFrom := regexp.MustCompile(`SSHD_OPTS=.*`)
	reTo := `SSHD_OPTS="-o X11UseLocalhost=no"`
	destFile := "/etc/default/ssh"

	f, err := os.OpenFile(destFile, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	replaced := reFrom.ReplaceAll(data, []byte(reTo))
	st, err := f.Stat()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(destFile, replaced, st.Mode())
}

func mountHomeDisk(ctx context.Context) error {
	f, err := os.OpenFile("/etc/fstab", os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	if bytes.Contains([]byte(homeDisk), data) {
		return nil
	}

	err = neco.StopService(ctx, "ssh")
	if err != nil {
		return err
	}

	err = neco.RetryWithSleep(ctx, retryCount, time.Second,
		func(ctx context.Context) error {
			active, err := neco.IsActiveService(ctx, "google-accounts-daemon")
			if err != nil {
				return err
			}
			if !active {
				return errors.New("google-accounts-daemon.service is not yet active")
			}
			return nil

		},
		func(err error) {
			log.Error("timeout for checking service is active", map[string]interface{}{
				log.FnError: err,
				"service":   "google-accounts-daemon.service",
			})
		},
	)
	if err != nil {
		return err
	}

	err = neco.StopService(ctx, "google-accounts-daemon")
	if err != nil {
		return err
	}

	err = well.CommandContext(ctx, "/sbin/dumpe2fs", "-h", homeDisk).Run()
	if err != nil {
		err := formatHomeDisk(ctx)
		if err != nil {
			return err
		}
	}

	_, err = io.WriteString(f, fmt.Sprintf("%s %s %s defaults 1 1", homeDisk, homeMountPoint, homeFSType))
	if err != nil {
		return err
	}

	err = syscall.Mount(homeDisk, homeMountPoint, homeFSType, syscall.MS_RELATIME, "")
	if err != nil {
		return err
	}

	err = neco.StartService(ctx, "google-accounts-daemon")
	if err != nil {
		return err
	}
	return neco.StartService(ctx, "ssh")
}

func formatHomeDisk(ctx context.Context) error {
	err := well.CommandContext(ctx, "/sbin/mkfs", "-t", homeFSType, homeDisk).Run()
	if err != nil {
		return err
	}

	err = syscall.Mount(homeDisk, "/mnt", homeFSType, syscall.MS_RELATIME, "")
	if err != nil {
		return err
	}

	render := func(srcFile, destFile string) error {
		f, err := os.Open(srcFile)
		if err != nil {
			return err
		}
		defer f.Close()

		data, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		st, err := f.Stat()
		if err != nil {
			return err
		}
		return ioutil.WriteFile(destFile, []byte(data), st.Mode())
	}

	src := homeMountPoint
	dest := "/mnt"
	err = filepath.Walk(src, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}

		target := filepath.Join(dest, rel)
		_, err = os.Stat(target)
		if err == nil {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(target, 0755)
		}

		return render(p, target)
	})
	if err != nil {
		return err
	}

	return syscall.Unmount("/mnt", syscall.MNT_FORCE)
}

func setupLocalSSD(ctx context.Context) error {
	err := well.CommandContext(ctx, "/sbin/mkfs", "-t", localSSDFSType, "-F", localSSDDisk).Run()
	if err != nil {
		return err
	}

	err = os.MkdirAll(localSSDMountPoint, 0755)
	if err != nil {
		return err
	}

	err = syscall.Mount(localSSDDisk, localSSDMountPoint, localSSDFSType, syscall.MS_RELATIME, "")
	if err != nil {
		return err
	}

	return os.Chmod(localSSDMountPoint, 0777)
}
