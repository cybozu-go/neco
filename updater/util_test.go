package updater

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/storage/test"
	"github.com/cybozu-go/well"
)

type Env struct {
	storage.Storage
	slack *SlackServer
	etcd  *clientv3.Client
	sess  *concurrency.Session

	env *well.Environment
}

func NewTestEnv(t *testing.T) *Env {
	ctx := context.Background()

	var err error

	e := new(Env)
	e.etcd = test.NewEtcdClient(t)
	e.sess, err = concurrency.NewSession(e.etcd)
	if err != nil {
		t.Fatal(err)
	}
	e.Storage = storage.NewStorage(e.etcd)
	e.slack = NewSlackServer()

	err = e.Storage.PutSlackNotification(ctx, e.slack.URL())
	if err != nil {
		t.Fatal(err)
	}

	return e
}

func (e *Env) Start() {
	e.env = well.NewEnvironment(context.Background())
	e.env.Go(func(ctx context.Context) error {
		server := Server{
			session: e.sess,
			storage: e.Storage,
			timeout: time.Second,
		}
		return server.Run(ctx)
	})
	e.env.Stop()
}

func (e *Env) WaitMessage() (Payload, error) {
	select {
	case msg, ok := <-e.slack.WatchMessage():
		if !ok {
			return msg, errors.New("slack server was closed")
		}
		return msg, nil
	case <-time.After(time.Second):
		return Payload{}, errors.New("time out")
	}
}

func (e *Env) Shutdown() {
	e.env.Cancel(nil)
	err := e.env.Wait()
	if err != nil {
		fmt.Println(err)
	}
	e.etcd.Close()
	e.slack.Close()
}

func PutRequest(t *testing.T, req neco.UpdateRequest) error {
	ctx := context.Background()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	st := storage.NewStorage(etcd)

	sess, err := concurrency.NewSession(etcd)
	if err != nil {
		return err
	}
	defer sess.Close()

	e := concurrency.NewElection(sess, storage.KeyUpdaterLeader)
	err = e.Campaign(ctx, "test")
	if err != nil {
		return err
	}
	leaderKey := e.Key()

	err = st.PutRequest(ctx, req, leaderKey)
	if err != nil {
		return err
	}

	return e.Resign(ctx)
}

type EtcdWatcher struct {
	t   *testing.T
	key string
	rev int64
}

func NewWatcher(t *testing.T, key string) (*EtcdWatcher, error) {
	etcd := test.NewEtcdClient(t)
	defer etcd.Close()

	resp, err := etcd.Get(context.Background(), key)
	if err != nil {
		return nil, err
	}
	var rev int64
	if resp.Count != 0 {
		rev = resp.Kvs[0].ModRevision
	}
	return &EtcdWatcher{t: t, key: key, rev: rev}, nil
}

func (w *EtcdWatcher) Wait() error {
	etcd := test.NewEtcdClient(w.t)
	defer etcd.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch := etcd.Watch(ctx, w.key, clientv3.WithRev(w.rev+1), clientv3.WithFilterDelete())
	resp := <-ch
	if resp.Err() != nil {
		return resp.Err()
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}
