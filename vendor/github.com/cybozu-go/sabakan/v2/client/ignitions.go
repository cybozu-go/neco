package client

import (
	"context"
)

// IgnitionsListIDs gets list of ignition template IDs for a role
func (c *Client) IgnitionsListIDs(ctx context.Context, role string) ([]string, error) {
	var ids []string
	err := c.getJSON(ctx, "ignitions/"+role, nil, &ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// IgnitionsGet gets an ignition template identified by role and id.
func (c *Client) IgnitionsGet(ctx context.Context, role, id string) (*IgnitionTemplate, error) {
	tmpl := &IgnitionTemplate{}
	err := c.getJSON(ctx, "ignitions/"+role+"/"+id, nil, tmpl)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

// IgnitionsSet puts an ignition template file
func (c *Client) IgnitionsSet(ctx context.Context, role, id string, tmpl *IgnitionTemplate) error {
	return c.sendRequestWithJSON(ctx, "PUT", "ignitions/"+role+"/"+id, tmpl)
}

// IgnitionsDelete deletes an ignition template specified by role and id
func (c *Client) IgnitionsDelete(ctx context.Context, role, id string) error {
	return c.sendRequest(ctx, "DELETE", "ignitions/"+role+"/"+id, nil)
}
