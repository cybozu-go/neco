package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/well"
	serf "github.com/hashicorp/serf/client"
	"github.com/prometheus/prom2json"
)

var (
	flagSabakanAddress = flag.String("sabakan-address", "http://localhost:10080", "sabakan address")
)

type machineStateSource struct {
	serial     string
	serfStatus serf.Member
	metrics    []*prom2json.Family
}

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	localHTTPClient := ext.LocalHTTPClient()
	sm := new(searchMachineResponse)
	sabakanEndpoint := path.Join(*flagSabakanAddress, "/graphql")
	well.Go(func(ctx context.Context) error {
		sm, err := getSabakanMachines(ctx, localHTTPClient, sabakanEndpoint)
		if err != nil {
			return err
		}
		return nil
	})
	well.Stop()
	err := well.Wait()
	if err != nil {
		return err
	}
	if len(sm.SearchMachines) == 0 {
		return errors.New("no machines found")
	}

	mss := make([]machineStateSource, len(sm.SearchMachines))

	_, err = getSerfStatus()
	if err != nil {
		return err
	}

	_, err = getMetrics(mcs)
	if err != nil {
		return err
	}

	for _, ms := range mss {
		well.Go(func(ctx context.Context) error {
			return setSabakanStates(ctx, ms)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}
