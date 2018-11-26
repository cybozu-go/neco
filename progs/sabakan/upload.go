package sabakan

import (
	"context"
	"net/http"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/sabakan/client"
	"github.com/cybozu-go/well"
)

const (
	sabakanEndpoint = "http://127.0.0.1:10080"
	imageOS         = "coreos"
)

// UploadContents upload contents to sabakan
func UploadContents(ctx context.Context) error {
	client, err := NewClient()
	if err != nil {
		return err
	}

	err = uploadOSImages(ctx)
	if err != nil {
		return err
	}

	return nil
}

type Client struct{}

// NewClient returns new sabakan client
func NewClient() (*Client, error) {
	err := client.Setup(sabakanEndpoint, &well.HTTPClient{
		Severity: log.LvDebug,
		Client:   &http.Client{},
	})
	if err != nil {
		return nil, err
	}

	return &Client{}, nil
}

// UploadOSImages uploads CoreOS images
func (c *Client) UploadOSImages(ctx context.Context) error {
	// TODO
	return nil
}

// UploadAssets uploads assets
func (c *Client) UploadAssets(ctx context.Context) error {
	// TODO
	return nil
}

// UploadIgnitions uploads ignitions
func (c *Client) UploadIgnitions(ctx context.Context) error {
	// TODO
	return nil
}
