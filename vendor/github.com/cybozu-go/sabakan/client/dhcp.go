package client

import (
	"context"

	"github.com/cybozu-go/sabakan"
)

// DHCPConfigGet retrieves DHCP configurations
func (c *Client) DHCPConfigGet(ctx context.Context) (*sabakan.DHCPConfig, error) {
	conf := new(sabakan.DHCPConfig)
	err := c.getJSON(ctx, "config/dhcp", nil, conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

// DHCPConfigSet sets DHCP configurations
func (c *Client) DHCPConfigSet(ctx context.Context, conf *sabakan.DHCPConfig) error {
	return c.sendRequestWithJSON(ctx, "PUT", "config/dhcp", conf)
}
