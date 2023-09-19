package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	sss "github.com/cybozu-go/neco/pkg/sabakan-state-setter"
	"github.com/cybozu-go/well"
)

var (
	flagConfigFile      = flag.String("config-file", "", "path of config file")
	flagEtcdSessionTTL  = flag.Duration("etcd-session-ttl", 1*time.Minute, "TTL of etcd session")
	flagInterval        = flag.Duration("interval", 1*time.Minute, "interval of scraping metrics")
	flagParallelSize    = flag.Int("parallel", 30, "The number of parallel execution of getting machines metrics")
	flagSabakanURL      = flag.String("sabakan-url", "http://localhost:10080", "sabakan URL")
	flagSabakanURLHTTPS = flag.String("sabakan-url-https", "https://localhost:10443", "sabakan TLS URL")
	flagSerfAddress     = flag.String("serf-address", "127.0.0.1:7373", "serf address")
)

func main() {
	flag.Parse()
	err := well.LogConfig{}.Apply()
	if err != nil {
		log.ErrorExit(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.ErrorExit(err)
	}

	etcdClient, err := neco.EtcdClient()
	if err != nil {
		log.ErrorExit(err)
	}
	defer etcdClient.Close()

	// Using well.Go for terminating this process when catche a signal.
	well.Go(func(ctx context.Context) error {
		ctr, err := sss.NewController(etcdClient, *flagSabakanURL, *flagSabakanURLHTTPS, *flagSerfAddress, *flagConfigFile, hostname, *flagInterval, *flagParallelSize, *flagEtcdSessionTTL)
		if err != nil {
			return fmt.Errorf("failed to create controller: %s", err.Error())
		}
		return ctr.Run(ctx)
	})
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.Error("error exit", map[string]interface{}{
			log.FnError: err,
		})
		os.Exit(1)
	}
	log.Info("exit", nil)
}
