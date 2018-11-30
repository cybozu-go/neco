package sabakan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/sabakan"
	"github.com/cybozu-go/sabakan/client"
)

const (
	endpoint = "http://127.0.0.1:10080"
	imageOS  = "coreos"
)

const retryCount = 5

// UploadContents upload contents to sabakan
func UploadContents(ctx context.Context, sabakanHTTP *http.Client, proxyHTTP *http.Client, version string) error {
	client, err := client.NewClient(endpoint, sabakanHTTP)
	if err != nil {
		return err
	}

	err = uploadOSImages(ctx, client, proxyHTTP)
	if err != nil {
		return err
	}

	err = uploadAssets(ctx, client)
	if err != nil {
		return err
	}

	err = uploadIgnitions(ctx, client, version)
	if err != nil {
		return err
	}

	return nil
}

func retry(ctx context.Context, f func(ctx context.Context) error) error {
	for i := 0; i < retryCount; i++ {
		err := f(ctx)
		if err == nil {
			return nil
		}
		log.Warn("sabakan: failed to ", map[string]interface{}{
			log.FnError: err,
		})
		err2 := neco.SleepContext(ctx, 10*time.Second)
		if err2 != nil {
			return err2
		}
	}
	if err != nil {
		return err
	}
}

// uploadOSImages uploads CoreOS images
func uploadOSImages(ctx context.Context, c *client.Client, p *http.Client) error {
	var index sabakan.ImageIndex
	var err error
	for i := 0; i < retryCount; i++ {
		index, err = c.ImagesIndex(ctx, imageOS)
		if err == nil {
			break
		}
		log.Warn("sabakan: failed to get index of CoreOS images", map[string]interface{}{
			log.FnError: err,
		})
		err2 := neco.SleepContext(ctx, 10*time.Second)
		if err2 != nil {
			return err2
		}
	}
	if err != nil {
		return err
	}

	version := neco.CurrentArtifacts.CoreOS.Version
	if len(index) != 0 && index[len(index)-1].ID == version {
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
	for i := 0; i < retryCount; i++ {
		err = kernelFile.Truncate(0)
		if err != nil {
			return err
		}
		_, err = kernelFile.Seek(0, 0)
		if err != nil {
			return err
		}
		kernelSize, err = downloadFile(ctx, p, kernelURL, kernelFile)
		if err == nil {
			break
		}
		log.Warn("sabakan: failed to fetch Container Linux kernel", map[string]interface{}{
			log.FnError: err,
			"url":       kernelURL,
		})
		err2 := neco.SleepContext(ctx, 10*time.Second)
		if err2 != nil {
			return err2
		}
	}
	if err != nil {
		return err
	}
	_, err = kernelFile.Seek(0, 0)
	if err != nil {
		return err
	}

	var initrdSize int64
	for i := 0; i < retryCount; i++ {
		err = initrdFile.Truncate(0)
		if err != nil {
			return err
		}
		_, err = initrdFile.Seek(0, 0)
		if err != nil {
			return err
		}
		initrdSize, err = downloadFile(ctx, p, initrdURL, initrdFile)
		if err == nil {
			break
		}
		log.Warn("sabakan: failed to fetch Container Linux initrd", map[string]interface{}{
			log.FnError: err,
			"url":       initrdURL,
		})
		err2 := neco.SleepContext(ctx, 10*time.Second)
		if err2 != nil {
			return err2
		}
	}
	if err != nil {
		return err
	}

	_, err = initrdFile.Seek(0, 0)
	if err != nil {
		return err
	}

	for i := 0; i < retryCount; i++ {
		err = c.ImagesUpload(ctx, imageOS, version, kernelFile, kernelSize, initrdFile, initrdSize)
		if err == nil {
			return nil

		}
		log.Warn("sabakan: failed to upload Container Linux", map[string]interface{}{
			log.FnError: err,
		})
		err2 := neco.SleepContext(ctx, 10*time.Second)
		if err2 != nil {
			return err2
		}
	}
	return err
}

var containers = []neco.ContainerImage{
	{
		Name:       "bird",
		Repository: "quay.io/cybozu/bird",
		Tag:        "2.0.2-7",
	},
	{
		Name:       "chrony",
		Repository: "quay.io/cybozu/chrony",
		Tag:        "3.3-4",
	},
}

func assetName(image neco.ContainerImage) string {
	return fmt.Sprintf("cybozu-%s-%s.aci", image.Name, image.Tag)
}

// uploadAssets uploads assets
func uploadAssets(ctx context.Context, c *client.Client) error {
	for _, container := range containers {
		err := neco.FetchContainer(ctx, container.FullName(), nil)
		if err != nil {
			return err
		}
		f, err := ioutil.TempFile("", "")
		if err != nil {
			return err
		}
		defer os.Remove(f.Name())
		err = neco.ExportContainer(ctx, container.FullName(), f.Name())
		if err != nil {
			return err
		}
		_, err = c.AssetsUpload(ctx, assetName(container), f.Name(), nil)
		if err != nil {
			return err
		}
	}
	// TODO
	return nil
}

// uploadIgnitions updates ignitions from local file
func uploadIgnitions(ctx context.Context, c *client.Client, id string) error {
	roles, err := getInstalledRoles()
	if err != nil {
		return err
	}

	for _, role := range roles {
		path := ignitionPath(role)

		newer := new(bytes.Buffer)
		err := client.AssembleIgnitionTemplate(path, newer)
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

func needIgnitionUpdate(ctx context.Context, c *client.Client, role, id string, newer string) (bool, error) {
	index, err := c.IgnitionsGet(ctx, role)
	if client.IsNotFound(err) {
		return true, nil
	}
	if err != nil {
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
