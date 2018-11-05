package main

import (
	"flag"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/updater"
	"github.com/cybozu-go/well"
	yaml "gopkg.in/yaml.v2"
)

var (
	flgConfig     = flag.String("config", "/etc/neco/config.yml", "Configuration file path.")
	flgSessionTTL = flag.String("session-ttl", "60s", "leader session's TTL")
)

func loadConfig(p string) (*etcdutil.Config, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := etcdutil.NewConfig(neco.NecoPrefix)
	err = yaml.NewDecoder(f).Decode(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func main() {
	flag.Parse()
	well.LogConfig{}.Apply()

	ttl, err := time.ParseDuration(*flgSessionTTL)
	if err != nil {
		log.ErrorExit(err)
	}
	cfg, err := loadConfig(*flgConfig)
	if err != nil {
		log.ErrorExit(err)
	}

	timeout, err := time.ParseDuration(cfg.Timeout)
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
	server := updater.NewServer(session, st, timeout)

	well.Go(server.Run)
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}
