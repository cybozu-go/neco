package sabakan

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/sabakan/client"
)

const (
	endpoint = "http://127.0.0.1:10080"
	imageOS  = "coreos"
)

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
	kernelSize, err := downloadFile(ctx, p, kernelURL, kernelFile)
	if err != nil {
		return err
	}

	initrdFile, err := ioutil.TempFile("", "initrd")
	if err != nil {
		return err
	}
	defer func() {
		initrdFile.Close()
		os.Remove(initrdFile.Name())
	}()
	initrdSize, err := downloadFile(ctx, p, initrdURL, initrdFile)
	if err != nil {
		return err
	}

	return c.ImagesUpload(ctx, imageOS, version, kernelFile, kernelSize, initrdFile, initrdSize)
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
