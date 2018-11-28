package client

import (
	"context"
	"path"
	"strings"

	"github.com/cybozu-go/sabakan"
)

// KernelParamsGet retrieves kernel parameters
func (c *Client) KernelParamsGet(ctx context.Context, os string) (sabakan.KernelParams, error) {
	body, status := c.getBytes(ctx, path.Join("kernel_params", os))
	if status != nil {
		return "", status
	}

	return sabakan.KernelParams(body), status
}

// KernelParamsSet sets kernel parameters
func (c *Client) KernelParamsSet(ctx context.Context, os string, params sabakan.KernelParams) error {
	r := strings.NewReader(string(params))
	return c.sendRequest(ctx, "PUT", path.Join("kernel_params", os), r)
}
