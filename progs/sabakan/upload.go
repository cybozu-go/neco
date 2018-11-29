package sabakan

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/sabakan/client"
)

const (
	endpoint = "http://127.0.0.1:10080"
	imageOS  = "coreos"
)

const retryCount = 5

// UploadContents upload contents to sabakan
func UploadContents(ctx context.Context, sabakanHTTP *http.Client, proxyHTTP *http.Client) error {
	client, err := client.NewClient(endpoint, sabakanHTTP)
	if err != nil {
		return err
	}

	err = UploadOSImages(ctx, client, proxyHTTP)
	if err != nil {
		return err
	}

	// UploadAssets
	// UploadIgnitions

	return nil
}

// UploadOSImages uploads CoreOS images
func UploadOSImages(ctx context.Context, c *client.Client, p *http.Client) error {
	index, err := c.ImagesIndex(ctx, imageOS)
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

// UploadAssets uploads assets
func UploadAssets(ctx context.Context) error {
	// TODO
	return nil
}

// UploadIgnitions uploads ignitions
func UploadIgnitions(ctx context.Context) error {
	// TODO
	return nil
}
