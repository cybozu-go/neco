package updater

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/google/go-cmp/cmp"
)

func testRunNoMembers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	e := NewTestEnv(t)

	w, err := NewWatcher(t, storage.KeyCurrent)
	if err != nil {
		t.Fatal(err)
	}

	e.Start()
	defer e.Shutdown()

	w.Wait()

	_, err = e.GetRequest(ctx)
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

	w, err := NewWatcher(t, storage.KeyCurrent)
	if err != nil {
		t.Fatal(err)
	}

	e.Start()
	defer e.Shutdown()

	w.Wait()

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

	msg, err := e.WaitMessage()
	if err != nil {
		t.Fatal(err)
	}
	if color := msg.Attachments[0].Color; color != ColorGood {
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

	w, err := NewWatcher(t, storage.KeyCurrent)
	if err != nil {
		t.Fatal(err)
	}

	e.Start()
	defer e.Shutdown()

	w.Wait()

	err = e.PutStatus(ctx, 0, neco.UpdateStatus{
		Version:  "1.0.0",
		Finished: true,
		Error:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := e.WaitMessage()
	if err != nil {
		t.Fatal(err)
	}
	if color := msg.Attachments[0].Color; color != ColorDanger {
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
	defer e.Shutdown()

	msg, err := e.WaitMessage()
	if err != nil {
		t.Fatal(err)
	}
	if color := msg.Attachments[0].Color; color != ColorDanger {
		t.Error("color != ColorDanger:", color)
	}
}

func testContinueUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	e := NewTestEnv(t)

	for _, lrn := range []int{0, 1, 2} {
		err := e.RegisterBootserver(ctx, lrn)
		if err != nil {
			t.Fatal(err)
		}
	}
	err := PutRequest(t, neco.UpdateRequest{
		Version:   "1.0.0",
		Servers:   []int{0, 1, 2},
		StartedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, lrn := range []int{0, 1} {
		err := e.PutStatus(ctx, lrn, neco.UpdateStatus{Version: "1.0.0", Finished: true})
		if err != nil {
			t.Fatal(err)
		}
	}

	e.Start()
	defer e.Shutdown()

	err = e.PutStatus(ctx, 2, neco.UpdateStatus{Version: "1.0.0", Finished: true})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := e.WaitMessage()
	if err != nil {
		t.Fatal(err)
	}
	if color := msg.Attachments[0].Color; color != ColorGood {
		t.Error("color != ColorGreen:", color)
	}
}

func testRun(t *testing.T) {
	t.Run("NoMembers", testRunNoMembers)
	t.Run("InitialUpdate", testRunInitialUpdate)
	t.Run("UpdateFailure", testRunUpdateFailure)
	t.Run("UpdateTimeout", testRunUpdateTimeout)
	t.Run("ContinueUpdate", testContinueUpdate)
}
func TestServer(t *testing.T) {
	t.Run("Run", testRun)
}
