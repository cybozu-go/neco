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

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/well"
	"github.com/rakyll/statik/fs"
)

const (
	assetDir = "/assets"
)

var (
	staticFiles = []string{
		"/etc/apt/apt.conf.d/20auto-upgrades",
		"/etc/bash_completion.d/rktutil",
		"/etc/containers/registries.conf",
		"/etc/profile.d/go.sh",
		"/usr/local/bin/podenter",
	}
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

	err = installSeaBIOS(client)
	if err != nil {
		return err
	}

	err = installGo(client)
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

	err = setupPodman()
	if err != nil {
		return err
	}

	err = dumpStaticFiles()
	if err != nil {
		return err
	}

	if project == "neco-test" {
		err = downloadAssets(client)
		if err != nil {
			return err
		}
	}
	return nil
}

func configureApt(ctx context.Context) error {
	err := neco.StopTimer(ctx, "apt-daily-upgrade")
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

func installSeaBIOS(client *http.Client) error {
	for _, url := range artifacts.seaBIOSURLs() {
		err := downloadFile(client, url, "/usr/share/seabios")
		if err != nil {
			return err
		}
	}

	return nil
}

func installGo(client *http.Client) error {
	resp, err := client.Get(artifacts.goURL())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return untargz(resp.Body, "/usr/local")
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

func setupPodman() error {
	return os.Symlink("/usr/lib/cri-o-runc/sbin/runc", "/usr/local/sbin/runc")
}

func dumpStaticFiles() error {
	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	for _, file := range staticFiles {
		err := copyStatic(statikFS, file)
		if err != nil {
			return err
		}
		log.Info("wrote", map[string]interface{}{
			"file": file,
		})
	}

	return nil
}

func copyStatic(fs http.FileSystem, fileName string) error {
	src, err := fs.Open(fileName)
	if err != nil {
		return err
	}
	defer src.Close()

	fi, err := src.Stat()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0755)
	if err != nil {
		return err
	}

	dst, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer dst.Close()

	err = dst.Chmod(fi.Mode())
	if err != nil {
		return err
	}

	_, err = io.Copy(dst, src)
	return err
}

func downloadAssets(client *http.Client) error {
	err := os.MkdirAll(assetDir, 0755)
	if err != nil {
		return err
	}

	// Download files
	for _, url := range artifacts.assetURLs() {
		err := downloadFile(client, url, assetDir)
		if err != nil {
			return err
		}
		log.Info("downloaded", map[string]interface{}{
			"url": url,
		})
	}

	// Decompress bzip2 archives
	for _, bz2file := range artifacts.bz2Files() {
		bz2, err := os.Open(filepath.Join(assetDir, bz2file))
		if err != nil {
			return err
		}
		defer func() {
			bz2.Close()
			os.Remove(bz2.Name())
		}()
		f := bzip2.NewReader(bz2)
		extName := strings.TrimRight(bz2.Name(), ".bz2")
		err = writeToFile(extName, f)
		if err != nil {
			return err
		}
		log.Info("decompressed", map[string]interface{}{
			"from": bz2.Name(),
			"to":   extName,
		})
	}

	return nil
}

func downloadFile(client *http.Client, url, destDir string) error {
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
