package main

import (
	"context"

	"github.com/cybozu-go/neco/ext"
	sabakan "github.com/cybozu-go/sabakan/v2"
	sabakanClient "github.com/cybozu-go/sabakan/v2/client"
)

func getSabakanMachines(ctx context.Context, address string) ([]sabakan.Machine, error) {
	saba, err := sabakanClient.NewClient(address, ext.LocalHTTPClient())
	if err != nil {
		return nil, err
	}

	params := map[string]string{}
	return saba.MachinesGet(ctx, params)
}

func setSabakanStates(ctx context.Context, ms machineStateSource) error {
	return nil
}
