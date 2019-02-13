package gcp

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/well"
)

// SetupVMXEnabled setup vmx-enabled instance
func SetupVMXEnabled(ctx context.Context, project string, option []string) error {
	err := configureApt(ctx)
	if err != nil {
		return err
	}

	err = configureProjectAtomic(ctx)
	if err != nil {
		return err
	}

	err = installAptPackages(ctx, option)
	if err != nil {
		return err
	}

	client := ext.LocalHTTPClient()

	err = installSeaBIOS(ctx, client)
	if err != nil {
		return err
	}

	err = installGo(ctx, client)
	if err != nil {
		return err
	}

	err = installDebianPackage(ctx, client, artifacts.rktURL())
	if err != nil {
		return err
	}

	err = installDebianPackage(ctx, client, artifacts.placematURL())
	if err != nil {
		return err
	}

	err = setupPodman(ctx)
	if err != nil {
		return err
	}

	if project == "neco-test" {
		err = downloadAssets(ctx, client)
		if err != nil {
			return err
		}
	}
	return nil
}

func configureApt(ctx context.Context) error {
	err := neco.WriteFile(aptConf, aptConfData)
	if err != nil {
		return err
	}

	err = neco.StopTimer(ctx, "apt-daily-upgrade")
	if err != nil {
		return err
	}
	err = neco.DisableTimer(ctx, "apt-daily-upgrade")
	if err != nil {
		return err
	}
	err = neco.StopService(ctx, "apt-daily-upgrade")
	if err != nil {
		return err
	}
	err = neco.StopTimer(ctx, "apt-daily")
	if err != nil {
		return err
	}
	err = neco.DisableTimer(ctx, "apt-daily")
	if err != nil {
		return err
	}
	err = neco.StopService(ctx, "apt-daily")
	if err != nil {
		return err
	}

	err = apt(ctx, "purge", "-y", "--autoremove", "unattended-upgrades")
	if err != nil {
		return err
	}
	err = apt(ctx, "update")
	if err != nil {
		return err
	}
	err = apt(ctx, "install", "-y", "apt-transport-https")
	if err != nil {
		return err
	}

	return nil
}

func apt(ctx context.Context, args ...string) error {
	return well.CommandContext(ctx, "apt-get", args...).Run()
}

func configureProjectAtomic(ctx context.Context) error {
	err := apt(ctx, "install", "-y", "software-properties-common", "dirmngr")
	if err != nil {
		return err
	}

	err = well.CommandContext(ctx, "apt-key", "adv", "--keyserver", "keyserver.ubuntu.com", "--recv", "7AD8C79D").Run()
	if err != nil {
		return err
	}

	return well.CommandContext(ctx, "add-apt-repository", "deb http://ppa.launchpad.net/projectatomic/ppa/ubuntu xenial main").Run()
}

func installAptPackages(ctx context.Context, optionalPackages []string) error {
	err := apt(ctx, "update")
	if err != nil {
		return err
	}

	args := []string{"install", "-y", "--no-install-recommends"}
	args = append(args, artifacts.debPackages...)
	err = apt(ctx, args...)
	if err != nil {
		return err
	}
	if len(optionalPackages) != 0 {
		args := []string{"install", "-y", "--no-install-recommends"}
		args = append(args, optionalPackages...)
		err = apt(ctx, args...)
		if err != nil {
			return err
		}
	}
	return apt(ctx, "clean")
}

func installSeaBIOS(ctx context.Context, client *http.Client) error {
	for _, url := range artifacts.seaBIOSURLs() {
		err := downloadFile(ctx, client, url, "/usr/share/seabios")
		if err != nil {
			return err
		}
	}

	return nil
}

func installGo(ctx context.Context, client *http.Client) error {
	resp, err := client.Get(artifacts.goURL())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return untargz(resp.Body, "/usr/local/go")
}

func untargz(r io.Reader, dst string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(dst, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			f.Close()
		}
	}
}

func writeToFile(p string, r io.Reader) error {
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()

	err = f.Chmod(0644)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, r)
	if err != nil {
		return err
	}

	return f.Sync()
}

func installDebianPackage(ctx context.Context, client *http.Client, url string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	command := []string{"sh", "-c", "dpkg -i " + f.Name() + " && rm " + f.Name()}
	return well.CommandContext(ctx, command[0], command[1:]...).Run()
}

func setupPodman(ctx context.Context) error {
	err := os.Symlink("/usr/lib/cri-o-runc/sbin/runc", "/usr/local/sbin/runc")
	if err != nil {
		return err
	}
	return neco.WriteFile(registriesConf, registriesConfData)
}

func downloadFile(ctx context.Context, client *http.Client, url, destDir string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(filepath.Join(destDir, filepath.Base(url)), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	return f.Sync()
}

func downloadAssets(ctx context.Context, client *http.Client) error {
	err := os.MkdirAll(assetDir, 0755)
	if err != nil {
		return err
	}

	// Download files
	for _, url := range artifacts.assetURLs() {
		err := downloadFile(ctx, client, url, assetDir)
		if err != nil {
			return err
		}
	}

	// Decompress bzip2 archives
	for _, file := range artifacts.bz2Files() {
		bz2, err := os.Open(file)
		if err != nil {
			return err
		}
		defer func() {
			bz2.Close()
			os.Remove(bz2.Name())
		}()
		f := bzip2.NewReader(bz2)
		extName := filepath.Join(assetDir, strings.TrimRight(bz2.Name(), ".bz2"))
		err = writeToFile(extName, f)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetupHostVM setup vmx-enabled instance
func SetupHostVM(ctx context.Context) error {
	return nil
}
