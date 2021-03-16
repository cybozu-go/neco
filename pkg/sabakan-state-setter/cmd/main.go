package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	sss "github.com/cybozu-go/neco/pkg/sabakan-state-setter"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
)

var (
	flagSabakanAddress = flag.String("sabakan-address", "http://localhost:10080", "sabakan address")
	flagSerfAddress    = flag.String("serf-address", "127.0.0.1:7373", "serf address")
	flagConfigFile     = flag.String("config-file", "", "path of config file")
	flagInterval       = flag.String("interval", "1m", "interval of scraping metrics")
	flagSessionTTL     = flag.String("session-ttl", "60s", "leader session's TTL")
	flagParallelSize   = flag.Int("parallel", 30, "parallel size")
)

func watchLeaderKey(ctx context.Context, s *concurrency.Session, leaderKey string) error {
	ch := s.Client().Watch(ctx, leaderKey, clientv3.WithFilterPut())
	for {
		select {
		case <-s.Done():
			return errors.New("session is closed")
		case resp, ok := <-ch:
			if !ok {
				return errors.New("watch is closed")
			}
			if resp.Err() != nil {
				return resp.Err()
			}
			for _, ev := range resp.Events {
				if ev.Type == clientv3.EventTypeDelete {
					return errors.New("leader key is deleted")
				}
			}
		}
	}
}

func main() {
	flag.Parse()
	sessTTL, err := time.ParseDuration(*flagSessionTTL)
	if err != nil {
		log.ErrorExit(err)
	}
	interval, err := time.ParseDuration(*flagInterval)
	if err != nil {
		log.ErrorExit(err)
	}
	err = well.LogConfig{}.Apply()
	if err != nil {
		log.ErrorExit(err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		log.ErrorExit(err)
	}

	ctr, err := sss.NewController(*flagSabakanAddress, *flagSerfAddress, *flagConfigFile, interval, *flagParallelSize)
	if err != nil {
		log.ErrorExit(err)
	}

	etcd, err := neco.EtcdClient()
	if err != nil {
		log.ErrorExit(err)
	}
	defer etcd.Close()

	sess, err := concurrency.NewSession(etcd, concurrency.WithTTL(int(sessTTL.Seconds())))
	if err != nil {
		log.ErrorExit(err)
	}
	defer sess.Close()

	e := concurrency.NewElection(sess, storage.KeySabakanStateSetterLeader)
	err = e.Campaign(context.Background(), hostname)
	if err != nil {
		log.ErrorExit(err)
	}
	leaderKey := e.Key()

	log.Info("I am the leader", map[string]interface{}{
		"session": sess.Lease(),
	})

	env := well.NewEnvironment(context.Background())
	env.Go(func(ctx context.Context) error {
		return ctr.RunPeriodically(ctx)
	})
	env.Go(func(ctx context.Context) error {
		return watchLeaderKey(ctx, sess, leaderKey)
	})
	env.Stop()
	err = env.Wait()

	// Release the leader before terminating this process.
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resignErr := e.Resign(ctxWithTimeout)
	if resignErr != nil {
		log.Error("failed to resign", map[string]interface{}{
			log.FnError: resignErr,
		})
	}

	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
	log.Info("exit", nil)
}
