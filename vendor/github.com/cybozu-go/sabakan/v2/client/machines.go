package client

import (
	"context"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/cybozu-go/sabakan/v2"
)

// MachinesGet get machine information from sabakan server
func (c *Client) MachinesGet(ctx context.Context, params map[string]string) ([]sabakan.Machine, error) {
	var machines []sabakan.Machine
	err := c.getJSON(ctx, "machines", params, &machines)
	if err != nil {
		return nil, err
	}
	return machines, nil
}

// MachinesCreate create machines information to sabakan server
func (c *Client) MachinesCreate(ctx context.Context, specs []*sabakan.MachineSpec) error {
	return c.sendRequestWithJSON(ctx, "POST", "machines", specs)
}

// MachinesRemove removes machine information from sabakan server
func (c *Client) MachinesRemove(ctx context.Context, serial string) error {
	return c.sendRequest(ctx, "DELETE", path.Join("machines", serial), nil)
}

// MachinesSetState set the state of the machine on sabakan server
func (c *Client) MachinesSetState(ctx context.Context, serial string, state string) error {
	r := strings.NewReader(state)
	return c.sendRequest(ctx, "PUT", "state/"+serial, r)
}

// MachinesGetState get the state of the machine from sabakan server
func (c *Client) MachinesGetState(ctx context.Context, serial string) (sabakan.MachineState, error) {
	req := c.newRequest(ctx, "GET", "state/"+serial, nil)
	resp, status := c.do(req)
	if status != nil {
		return "", status
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return sabakan.MachineState(data), nil
}

// MachinesSetLabel adds or updates a label for a machine on sabakan server.
func (c *Client) MachinesSetLabel(ctx context.Context, serial string, label, value string) error {
	r := strings.NewReader(value)
	return c.sendRequest(ctx, "PUT", path.Join("labels", serial, label), r)
}

// MachinesRemoveLabel removes a label from a machine on sabakan server.
func (c *Client) MachinesRemoveLabel(ctx context.Context, serial string, label string) error {
	return c.sendRequest(ctx, "DELETE", path.Join("labels", serial, label), nil)
}

// MachinesSetRetireDate set the retire date of the machine.
func (c *Client) MachinesSetRetireDate(ctx context.Context, serial string, date time.Time) error {
	input := strings.NewReader(date.Format(time.RFC3339))
	return c.sendRequest(ctx, "PUT", "retire-date/"+serial, input)
}
