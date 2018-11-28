package client

import (
	"context"
	"fmt"
	"io"
	"path"
)

// IgnitionsGet gets ignition template metadata list of the specified role
func (c *Client) IgnitionsGet(ctx context.Context, role string) ([]map[string]string, error) {
	var metadata []map[string]string
	err := c.getJSON(ctx, "ignitions/"+role, nil, &metadata)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

// IgnitionsCat gets an ignition template for the role an id
func (c *Client) IgnitionsCat(ctx context.Context, role, id string, w io.Writer) error {
	req := c.newRequest(ctx, "GET", path.Join("ignitions", role, id), nil)
	resp, status := c.do(req)
	if status != nil {
		return status
	}
	defer resp.Body.Close()

	_, err := io.Copy(w, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

// IgnitionsSet posts an ignition template file
func (c *Client) IgnitionsSet(ctx context.Context, role string, fname string, meta map[string]string) error {
	tmpl, err := generateIgnitionYAML(fname)
	if err != nil {
		return err
	}
	req := c.newRequest(ctx, "POST", "ignitions/"+role, tmpl)
	for k, v := range meta {
		req.Header.Set(fmt.Sprintf("X-Sabakan-Ignitions-%s", k), v)
	}
	resp, status := c.do(req)
	if status != nil {
		return status
	}
	resp.Body.Close()

	return nil
}

// IgnitionsDelete deletes an ignition template specified by role and id
func (c *Client) IgnitionsDelete(ctx context.Context, role, id string) error {
	return c.sendRequest(ctx, "DELETE", path.Join("ignitions", role, id), nil)
}
