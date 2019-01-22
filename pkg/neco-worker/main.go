package main

import (
	"context"
	"flag"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
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

	version, err := neco.GetDebianVersion(neco.NecoPackageName)
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
		// NOTE: hack for github.com/containers/image to set HTTP proxy
		st := storage.NewStorage(ec)
		proxy, err := st.GetProxyConfig(ctx)
		if err != nil && err != storage.ErrNotFound {
			return err
		}
		if len(proxy) != 0 {
			os.Setenv("http_proxy", proxy)
			os.Setenv("https_proxy", proxy)
		}

		op, err := worker.NewOperator(ctx, ec, mylrn)
		if err != nil {
			return err
		}
		w := worker.NewWorker(ec, op, version, mylrn)
		return w.Run(ctx)
	})
	well.Go(storage.NewStorage(ec).WaitConfigChange)
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}
