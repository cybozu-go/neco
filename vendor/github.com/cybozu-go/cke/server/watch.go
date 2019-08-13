package server

import (
	"context"
	"strings"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/cke"
)

func initStateless(ctx context.Context, etcd *clientv3.Client, ch chan<- struct{}) (int64, error) {
	defer func() {
		// notify the caller of the readiness
		ch <- struct{}{}
	}()

	resp, err := etcd.Get(ctx, cke.KeyVault)
	if err != nil {
		return 0, err
	}
	rev := resp.Header.Revision

	if len(resp.Kvs) == 1 {
		err = cke.ConnectVault(ctx, resp.Kvs[0].Value)
		if err != nil {
			return 0, err
		}
	}

	return rev, nil
}

func startWatcher(ctx context.Context, etcd *clientv3.Client, ch chan<- struct{}) error {
	rev, err := initStateless(ctx, etcd, ch)
	if err != nil {
		return err
	}

	wch := etcd.Watch(ctx, "", clientv3.WithPrefix(), clientv3.WithRev(rev+1))
	for resp := range wch {
		for _, ev := range resp.Events {
			if ev.Type != clientv3.EventTypePut {
				continue
			}

			key := string(ev.Kv.Key)
			switch {
			case key == cke.KeyCluster || strings.HasPrefix(key, cke.KeyResourcePrefix):
				select {
				case ch <- struct{}{}:
				default:
				}
			case key == cke.KeyVault:
				err = cke.ConnectVault(ctx, ev.Kv.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
