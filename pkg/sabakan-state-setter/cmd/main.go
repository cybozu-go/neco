package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	sss "github.com/cybozu-go/neco/pkg/sabakan-state-setter"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

var (
	flagSabakanAddress = flag.String("sabakan-address", "http://localhost:10080", "sabakan address")
	flagConfigFile     = flag.String("config-file", "", "path of config file")
	flagInterval       = flag.String("interval", "1m", "interval of scraping metrics")
	flagParallelSize   = flag.Int("parallel", 30, "parallel size")
)

func main() {
	flag.Parse()
	err := well.LogConfig{}.Apply()
	if err != nil {
		log.ErrorExit(err)
	}

	ctr, err := sss.NewController(context.Background(), *flagSabakanAddress, *flagConfigFile, *flagInterval, *flagParallelSize)
	if err != nil {
		log.ErrorExit(err)
	}
	well.Go(func(ctx context.Context) error {
		return ctr.RunPeriodically(ctx)
	})
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		fmt.Println(err)
		os.Exit(1)
	}
}
