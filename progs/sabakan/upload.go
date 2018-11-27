package sabakan

import (
	"context"
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
		return nil, err
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
func UploadOSImages(ctx context.Context, c client.Client, p *http.Client) error {
	index, err := c.ImagesIndex(ctx, imageOS)
	if err != nil {
		return nil
	}

	if len(index) != 0 && index[len(index)-1] == neco.CurrentArtifacts.CoreOS.Version {
		return nil
	}

	kernel, initrd := neco.CurrentArtifacts.CoreOS.URLs()
	version := neco.CurrentArtifacts.CoreOS.Version

	req, err := http.NewRequest("GET", kernel, nil)
	if err != nil {
		return err
	}
	resp, err := p.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
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
	initrdBody := resp.Body
	initrdSize := resp.ContentLength

	return c.ImagesUpload(imageOS, version, kernelBody, kernelSize, initrdBody, initrdSize)
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
