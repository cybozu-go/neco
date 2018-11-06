package main

import (
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

	w, err := worker.NewWorker(ec)
	if err != nil {
		log.ErrorExit(err)
	}

	well.Go(w.Run)
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}
