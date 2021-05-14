package cmd

import (
	"context"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/sabakan/v2"
	"github.com/cybozu-go/sabakan/v2/client"
	"github.com/spf13/cobra"
)

// sabakanMachinesGetOpts is a struct to receive option values for `sabactl machines get`-like options
type sabakanMachinesGetOpts struct {
	params map[string]*string
}

// addSabapanMachinesGetOpts adds flags for `sabactl machines get`-like options to cobra.Command
func addSabakanMachinesGetOpts(cmd *cobra.Command, opts *sabakanMachinesGetOpts) {
	getOpts := map[string]string{
		"serial":   "Serial name",
		"rack":     "Rack name",
		"role":     "Role name",
		"labels":   "Label name and value (--labels key=val,...)",
		"ipv4":     "IPv4 address",
		"ipv6":     "IPv6 address",
		"bmc-type": "BMC type",
		"state":    "State",
	}
	opts.params = make(map[string]*string)
	for k, v := range getOpts {
		val := new(string)
		opts.params[k] = val
		cmd.Flags().StringVar(val, k, "", v)
	}
}

// sabakanMachinesGet does the same as `sabactl machines get`
func sabakanMachinesGet(ctx context.Context, opts *sabakanMachinesGetOpts) ([]sabakan.Machine, error) {
	params := make(map[string]string)
	for k, v := range opts.params {
		params[k] = *v
	}
	c, err := client.NewClient(neco.SabakanLocalEndpoint, httpClient.Client)
	if err != nil {
		return nil, err
	}
	return c.MachinesGet(ctx, params)
}
