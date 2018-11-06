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
			Version: "1.0.0",
			Cond:    neco.CondComplete,
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
		Version: "1.0.0",
		Cond:    neco.CondAbort,
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

func testRunContinueUpdate(t *testing.T) {
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
		err := e.PutStatus(ctx, lrn, neco.UpdateStatus{
			Version: "1.0.0",
			Cond:    neco.CondComplete,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	e.Start()
	defer e.Shutdown()

	err = e.PutStatus(ctx, 2, neco.UpdateStatus{
		Version: "1.0.0",
		Cond:    neco.CondComplete,
	})
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

func testRunStopAndRetry(t *testing.T) {
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
		Stop:      true,
		StartedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(t, storage.KeyCurrent)
	if err != nil {
		t.Fatal(err)
	}

	e.Start()
	defer e.Shutdown()

	err = w.Wait()
	if err != context.DeadlineExceeded {
		t.Fatal("err != context.DeadlineExceeded:", err)
	}

	err = e.Storage.ClearStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = w.Wait()
	if err != nil {
		t.Fatal(err)
	}
}

func testRunNewMembers(t *testing.T) {
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
		Stop:      false,
		StartedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, lrn := range []int{0, 1, 2} {
		err := e.PutStatus(ctx, lrn, neco.UpdateStatus{Version: "1.0.0", Finished: true})
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

	_, err = e.WaitMessage()
	if err != nil {
		t.Fatal(err)
	}

	err = w.Wait()
	if err != context.DeadlineExceeded {
		t.Fatal("err != context.DeadlineExceeded:", err)
	}

	err = e.RegisterBootserver(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}

	err = w.Wait()
	if err != nil {
		t.Fatal("unexpected DeadlineExceeded or error:", err)
	}
}

func testRun(t *testing.T) {
	t.Run("NoMembers", testRunNoMembers)
	t.Run("InitialUpdate", testRunInitialUpdate)
	t.Run("UpdateFailure", testRunUpdateFailure)
	t.Run("UpdateTimeout", testRunUpdateTimeout)
	t.Run("ContinueUpdate", testRunContinueUpdate)
	t.Run("StopAndRetry", testRunStopAndRetry)
	t.Run("NewMembers", testRunNewMembers)
}
func TestServer(t *testing.T) {
	t.Run("Run", testRun)
}
