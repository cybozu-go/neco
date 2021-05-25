package main

import (
	"context"
	"flag"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/updater"
	"github.com/cybozu-go/well"
	"go.etcd.io/etcd/clientv3/concurrency"
)

var (
	flgSessionTTL = flag.String("session-ttl", "60s", "leader session's TTL")
)

func main() {
	flag.Parse()
	well.LogConfig{}.Apply()

	ttl, err := time.ParseDuration(*flgSessionTTL)
	if err != nil {
		log.ErrorExit(err)
	}

	etcd, err := neco.EtcdClient()
	if err != nil {
		log.ErrorExit(err)
	}
	defer etcd.Close()

	session, err := concurrency.NewSession(etcd, concurrency.WithTTL(int(ttl.Seconds())))
	if err != nil {
		log.ErrorExit(err)
	}
	defer session.Close()

	st := storage.NewStorage(etcd)

	well.Go(func(ctx context.Context) error {
		notifier, err := ext.NewNotifier(ctx, st)
		if err != nil {
			return err
		}
		server := updater.NewServer(session, st, notifier)
		return server.Run(ctx)
	})
	well.Go(st.WaitConfigChange)
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}
