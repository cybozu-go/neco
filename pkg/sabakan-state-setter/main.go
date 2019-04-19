package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	sabakan "github.com/cybozu-go/sabakan/v2"
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
	var mcs []sabakan.Machine
	well.Go(func(ctx context.Context) error {
		_, err := getSabakanMachines(ctx, *flagSabakanAddress)
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

	mss := make([]machineStateSource, len(mcs))

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
