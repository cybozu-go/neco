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

	well.Go(func(ctx context.Context) error {
		w, err := worker.NewWorker(ctx, ec)
		if err != nil {
			return err
		}
		return w.Run(ctx)
	})
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}
