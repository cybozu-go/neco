package main

import (
	"flag"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/updater"
	"github.com/cybozu-go/well"
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

	st := storage.NewStorage(etcd)
	server := updater.NewServer(session, st, 2*time.Second)

	well.Go(server.Run)
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}
