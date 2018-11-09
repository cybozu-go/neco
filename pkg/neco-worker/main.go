package main

import (
	"context"
	"flag"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/worker"
	"github.com/cybozu-go/well"
)

func main() {
	flag.Parse()
	well.LogConfig{}.Apply()

	ec, err := neco.EtcdClient()
	if err != nil {
		log.ErrorExit(err)
	}
	defer ec.Close()

	version, err := worker.GetDebianVersion(neco.NecoPackageName)
	if err != nil {
		log.ErrorExit(err)
	}
	if len(version) > 0 {
		log.Info("neco package version", map[string]interface{}{
			"version": version,
		})
	} else {
		log.Warn("no neco package", nil)
	}
	mylrn, err := neco.MyLRN()
	if err != nil {
		log.ErrorExit(err)
	}

	well.Go(func(ctx context.Context) error {
		op, err := worker.NewOperator(ctx, ec, mylrn)
		if err != nil {
			return err
		}
		w := worker.NewWorker(ec, op, version, mylrn)
		return w.Run(ctx)
	})
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}
