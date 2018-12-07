package sabakan

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	sabakan "github.com/cybozu-go/sabakan/client"
)

const (
	imageOS = "coreos"
)

const retryCount = 5

// UploadContents upload contents to sabakan
func UploadContents(ctx context.Context, sabakanHTTP *http.Client, proxyHTTP *http.Client, version string, auth neco.DockerAuth) error {
	client, err := sabakan.NewClient(neco.SabakanLocalEndpoint, sabakanHTTP)
	if err != nil {
		return err
	}

	err = uploadOSImages(ctx, client, proxyHTTP)
	if err != nil {
		return err
	}

	err = uploadAssets(ctx, client, auth)
	if err != nil {
		return err
	}

	err = uploadIgnitions(ctx, client, version)
	if err != nil {
		return err
	}

	return nil
}

// uploadOSImages uploads CoreOS images
func uploadOSImages(ctx context.Context, c *sabakan.Client, p *http.Client) error {
	var index sabakan.ImageIndex
	err := neco.RetryWithSleep(ctx, retryCount, 10*time.Second,
		func(ctx context.Context) error {
			var err error
			index, err = c.ImagesIndex(ctx, imageOS)
			return err
		},
		func(err error) {
			log.Warn("sabakan: failed to get index of CoreOS images", map[string]interface{}{
				log.FnError: err,
			})
		},
	)
	if err != nil {
		return err
	}

	version := neco.CurrentArtifacts.CoreOS.Version
	if len(index) != 0 && index[len(index)-1].ID == version {
		// already uploaded
		return nil
	}

	kernelURL, initrdURL := neco.CurrentArtifacts.CoreOS.URLs()

	kernelFile, err := ioutil.TempFile("", "kernel")
	if err != nil {
		return err
	}
	defer func() {
		kernelFile.Close()
		os.Remove(kernelFile.Name())
	}()
	initrdFile, err := ioutil.TempFile("", "initrd")
	if err != nil {
		return err
	}
	defer func() {
		initrdFile.Close()
		os.Remove(initrdFile.Name())
	}()

	var kernelSize int64
	err = neco.RetryWithSleep(ctx, retryCount, 10*time.Second,
		func(ctx context.Context) error {
			err := kernelFile.Truncate(0)
			if err != nil {
				return err
			}
			_, err = kernelFile.Seek(0, 0)
			if err != nil {
				return err
			}
			kernelSize, err = downloadFile(ctx, p, kernelURL, kernelFile)
			return err
		},
		func(err error) {
			log.Warn("sabakan: failed to fetch Container Linux kernel", map[string]interface{}{
				log.FnError: err,
				"url":       kernelURL,
			})
		},
	)
	if err != nil {
		return err
	}
	_, err = kernelFile.Seek(0, 0)
	if err != nil {
		return err
	}

	var initrdSize int64
	err = neco.RetryWithSleep(ctx, retryCount, 10*time.Second,
		func(ctx context.Context) error {
			err := initrdFile.Truncate(0)
			if err != nil {
				return err
			}
			_, err = initrdFile.Seek(0, 0)
			if err != nil {
				return err
			}
			initrdSize, err = downloadFile(ctx, p, initrdURL, initrdFile)
			return err
		},
		func(err error) {
			log.Warn("sabakan: failed to fetch Container Linux initrd", map[string]interface{}{
				log.FnError: err,
				"url":       initrdURL,
			})
		},
	)

	if err != nil {
		return err
	}

	_, err = initrdFile.Seek(0, 0)
	if err != nil {
		return err
	}

	err = neco.RetryWithSleep(ctx, retryCount, 10*time.Second,
		func(ctx context.Context) error {
			return c.ImagesUpload(ctx, imageOS, version, kernelFile, kernelSize, initrdFile, initrdSize)
		},
		func(err error) {
			log.Warn("sabakan: failed to upload Container Linux", map[string]interface{}{
				log.FnError: err,
			})
		},
	)
	return err
}

// uploadAssets uploads assets
func uploadAssets(ctx context.Context, c *sabakan.Client, auth neco.DockerAuth) error {
	// Upload bird and chrony with ubuntu-debug
	for _, img := range neco.SystemContainers {
		err := uploadSystemImageAssets(ctx, img, c)
		if err != nil {
			return err
		}
	}

	// Upload other images
	var fetches []neco.ContainerImage
	for _, name := range []string{"serf", "omsa", "coil"} {
		img, err := neco.CurrentArtifacts.FindContainerImage(name)
		if err != nil {
			return err
		}
		fetches = append(fetches, img)
	}
	for _, img := range fetches {
		err := UploadImageAssets(ctx, img, c, auth)
		if err != nil {
			return err
		}
	}

	// Upload sabakan-cryptsetup with version name
	img, err := neco.CurrentArtifacts.FindContainerImage("sabakan")
	if err != nil {
		return err
	}
	name := neco.CryptsetupAssetName(img)
	need, err := needAssetUpload(ctx, name, c)
	if err != nil {
		return err
	}
	if !need {
		return nil
	}
	err = neco.RetryWithSleep(ctx, retryCount, 10*time.Second,
		func(ctx context.Context) error {
			_, err := c.AssetsUpload(ctx, name, neco.SabakanCryptsetupPath, nil)
			return err

		},
		func(err error) {
			log.Warn("sabakan: failed to upload asset", map[string]interface{}{
				log.FnError: err,
				"name":      name,
				"source":    neco.SabakanCryptsetupPath,
			})
		},
	)
	return err
}

func uploadSystemImageAssets(ctx context.Context, img neco.ContainerImage, c *sabakan.Client) error {
	name := neco.ACIAssetName(img)
	need, err := needAssetUpload(ctx, name, c)
	if err != nil {
		return err
	}
	if !need {
		return nil
	}

	err = neco.RetryWithSleep(ctx, retryCount, 10*time.Second,
		func(ctx context.Context) error {
			_, err := c.AssetsUpload(ctx, name, neco.SystemImagePath(img), nil)
			return err

		},
		func(err error) {
			log.Warn("sabakan: failed to upload asset", map[string]interface{}{
				log.FnError: err,
				"name":      name,
				"source":    neco.SystemImagePath(img),
			})
		},
	)
	return err
}

// UploadImageAssets upload docker container image as sabakan assets.
func UploadImageAssets(ctx context.Context, img neco.ContainerImage, c *sabakan.Client, auth neco.DockerAuth) error {
	name := neco.ImageAssetName(img)
	need, err := needAssetUpload(ctx, name, c)
	if err != nil {
		return err
	}
	if !need {
		return nil
	}

	d, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(d)

	archive := filepath.Join(d, name)
	err = neco.FetchDockerImageAsArchive(ctx, img, archive, auth)
	if err != nil {
		return err
	}

	err = neco.RetryWithSleep(ctx, retryCount, 10*time.Second,
		func(ctx context.Context) error {
			_, err := c.AssetsUpload(ctx, name, archive, nil)
			return err

		},
		func(err error) {
			log.Warn("sabakan: failed to upload asset", map[string]interface{}{
				log.FnError: err,
				"name":      name,
				"source":    archive,
			})
		},
	)
	return err
}

func needAssetUpload(ctx context.Context, name string, c *sabakan.Client) (bool, error) {
	_, err := c.AssetsInfo(ctx, name)
	if err == nil {
		return false, nil
	}
	if sabakan.IsNotFound(err) {
		return true, nil
	}
	return false, err
}

// uploadIgnitions updates ignitions from local file
func uploadIgnitions(ctx context.Context, c *sabakan.Client, id string) error {
	roles, err := getInstalledRoles()
	if err != nil {
		return err
	}

	for _, role := range roles {
		path := ignitionPath(role)

		newer := new(bytes.Buffer)
		err := sabakan.AssembleIgnitionTemplate(path, newer)
		if err != nil {
			return err
		}

		need, err := needIgnitionUpdate(ctx, c, role, id, newer.String())
		if err != nil {
			return err
		}
		if !need {
			continue
		}
		err = c.IgnitionsSet(ctx, role, id, newer, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func needIgnitionUpdate(ctx context.Context, c *sabakan.Client, role, id string, newer string) (bool, error) {
	index, err := c.IgnitionsGet(ctx, role)
	if err != nil {
		if sabakan.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	latest := index[len(index)-1].ID
	if latest == id {
		return false, nil
	}

	current := new(bytes.Buffer)
	err = c.IgnitionsCat(ctx, role, latest, current)
	if err != nil {
		return false, err
	}
	return current.String() != newer, nil
}

func getInstalledRoles() ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(neco.IgnitionDirectory, "*", "site.yml"))
	if err != nil {
		return nil, err
	}
	for i, path := range paths {
		paths[i] = filepath.Base(filepath.Dir(path))
	}
	return paths, nil
}

func ignitionPath(role string) string {
	return filepath.Join(neco.IgnitionDirectory, role, "site.yml")
}

func downloadFile(ctx context.Context, p *http.Client, url string, w io.Writer) (int64, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(ctx)
	resp, err := p.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.ContentLength <= 0 {
		return 0, errors.New("unknown content-length")
	}
	return io.Copy(w, resp.Body)
}
