package sabakan

import (
	"context"
	"errors"
	"net/http"

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
		return nil
	}

	version := neco.CurrentArtifacts.CoreOS.Version
	if len(index) != 0 && index[len(index)-1].ID == version {
		return nil
	}

	kernel, initrd := neco.CurrentArtifacts.CoreOS.URLs()

	req, err := http.NewRequest("GET", kernel, nil)
	if err != nil {
		return err
	}
	resp, err := p.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.ContentLength <= 0 {
		return errors.New("unknown content-length")
	}
	kernelBody := resp.Body
	kernelSize := resp.ContentLength

	req, err = http.NewRequest("GET", initrd, nil)
	if err != nil {
		return err
	}
	resp, err = p.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.ContentLength <= 0 {
		return errors.New("unknown content-length")
	}
	initrdBody := resp.Body
	initrdSize := resp.ContentLength

	return c.ImagesUpload(ctx, imageOS, version, kernelBody, kernelSize, initrdBody, initrdSize)
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
