package updater

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/storage/test"
	"github.com/cybozu-go/neco/updater/mock"
	"github.com/cybozu-go/well"
	"github.com/google/go-cmp/cmp"
)

func testRunNoMembers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	e := NewTestEnv(t)
	e.Start()
	defer e.Close()

	time.Sleep(1000 * time.Millisecond)

	_, err := e.GetRequest(ctx)
	if err != storage.ErrNotFound {
		t.Error("err != ErrNotFound: ", err)
	}
}

func testRunInitialUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	e := NewTestEnv(t)

	for _, lrn := range []int{0, 1, 2} {
		err := e.RegisterBootserver(ctx, lrn)
		if err != nil {
			t.Fatal(err)
		}
	}

	e.Start()
	defer e.Close()
	time.Sleep(10 * time.Millisecond)

	req, err := e.GetRequest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sort.Ints(req.Servers)
	req2 := &neco.UpdateRequest{Version: "1.0.0", Servers: []int{0, 1, 2}, StartedAt: req.StartedAt}
	if !cmp.Equal(req, req2) {
		t.Error(`!cmp.Equal(req, req2)`, req)
	}

	for _, lrn := range []int{0, 1, 2} {
		err = e.PutStatus(ctx, lrn, neco.UpdateStatus{
			Version:  "1.0.0",
			Finished: true,
			Error:    false,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	msgs := e.SlackMessages()

	if len(msgs) != 1 {
		t.Fatal("len(msgs) != 1:", len(msgs))
	}
	if len(msgs[0].Attachments) != 1 {
		t.Fatal("len(msgs[0].Attachment) != 1:", len(msgs[0].Attachments))
	}
	if color := msgs[0].Attachments[0].Color; color != ColorGood {
		t.Error("color != ColorGood:", color)
	}
}

func testRunUpdateFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	e := NewTestEnv(t)

	for _, lrn := range []int{0, 1, 2} {
		err := e.RegisterBootserver(ctx, lrn)
		if err != nil {
			t.Fatal(err)
		}
	}

	e.Start()
	defer e.Close()
	time.Sleep(10 * time.Millisecond)

	err := e.PutStatus(ctx, 0, neco.UpdateStatus{
		Version:  "1.0.0",
		Finished: true,
		Error:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	msgs := e.SlackMessages()

	if len(msgs) != 1 {
		t.Fatal("len(msgs) != 1:", len(msgs))
	}
	if len(msgs[0].Attachments) != 1 {
		t.Fatal("len(msgs[0].Attachment) != 1:", len(msgs[0].Attachments))
	}
	if color := msgs[0].Attachments[0].Color; color != ColorDanger {
		t.Error("color != ColorDanger:", color)
	}
}

func testRunUpdateTimeout(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	e := NewTestEnv(t)

	for _, lrn := range []int{0, 1, 2} {
		err := e.RegisterBootserver(ctx, lrn)
		if err != nil {
			t.Fatal(err)
		}
	}

	err := e.PutWorkerTimeout(ctx, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	e.Start()
	defer e.Close()
	time.Sleep(100 * time.Millisecond)

	msgs := e.SlackMessages()

	if len(msgs) != 1 {
		t.Fatal("len(msgs) != 1:", len(msgs))
	}
	if len(msgs[0].Attachments) != 1 {
		t.Fatal("len(msgs[0].Attachment) != 1:", len(msgs[0].Attachments))
	}
	if color := msgs[0].Attachments[0].Color; color != ColorDanger {
		t.Error("color != ColorDanger:", color)
	}
}

func testRun(t *testing.T) {
	t.Run("NoMembers", testRunNoMembers)
	t.Run("InitialUpdate", testRunInitialUpdate)
	t.Run("UpdateFailure", testRunUpdateFailure)
	t.Run("UpdateTimeout", testRunUpdateTimeout)
}
func TestServer(t *testing.T) {
	t.Run("Run", testRun)
}

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

func (e *Env) SlackMessages() []Payload {
	return e.slack.Messages()
}

func (e *Env) Start() {
	e.env = well.NewEnvironment(context.Background())
	e.env.Go(func(ctx context.Context) error {
		server := Server{
			session: e.sess,
			storage: e.Storage,
			timeout: time.Second,
			checker: mock.ReleaseChecker{Version: "1.0.0"},
		}
		server.Run(ctx)

		e.etcd.Close()
		e.slack.Close()
		return nil
	})
	e.env.Stop()
}

func (e *Env) Close() {
	e.env.Cancel(nil)
	e.env.Wait()
}
