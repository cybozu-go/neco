package client

import (
	"bytes"
	"context"
	"path"
)

// CryptsGet gets an encryption key from sabakan server.
func (c *Client) CryptsGet(ctx context.Context, serial, device string) ([]byte, error) {
	return c.getBytes(ctx, path.Join("crypts", serial, device))
}

// CryptsPut puts an encryption key to sabakan server.
func (c *Client) CryptsPut(ctx context.Context, serial, device string, key []byte) error {
	r := bytes.NewReader(key)
	return c.sendRequest(ctx, "PUT", path.Join("crypts", serial, device), r)
}

// CryptsDelete removes all encryption keys of the machine specified by serial.
func (c *Client) CryptsDelete(ctx context.Context, serial string) error {
	return c.sendRequest(ctx, "DELETE", path.Join("crypts", serial), nil)
}
