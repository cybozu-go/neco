package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/storage/test"
	"github.com/cybozu-go/well"
	"github.com/google/go-cmp/cmp"
)

var errTest = errors.New("test error")

type mockOp struct {
	FailNecoUpdate bool
	FailAt         int

	NecoUpdated bool
	Step        int
	Req         *neco.UpdateRequest
}

// If failNeco is true, UpdateNeco() returns an error.
// If failAt is > 0, RunStep() returns an error at step == failAt.
func newMock(failNeco bool, failAt int) *mockOp {
	return &mockOp{
		FailNecoUpdate: failNeco,
		FailAt:         failAt,
	}
}

func expect(necoUpdated bool, step int, req *neco.UpdateRequest) *mockOp {
	return &mockOp{
		NecoUpdated: necoUpdated,
		Step:        step,
		Req:         req,
	}
}

func (op *mockOp) Equal(expected *mockOp) bool {
	if op.NecoUpdated != expected.NecoUpdated {
		return false
	}
	if op.Step != expected.Step {
		return false
	}
	return cmp.Equal(op.Req, expected.Req)
}

func (op *mockOp) UpdateNeco(ctx context.Context, req *neco.UpdateRequest) error {
	op.Req = req
	if op.FailNecoUpdate {
		return errTest
	}
	op.NecoUpdated = true
	return nil
}

func (op *mockOp) FinalStep() int {
	return 2
}

func (op *mockOp) RunStep(ctx context.Context, req *neco.UpdateRequest, step int) error {
	op.Step = step
	op.Req = req
	if op.FailAt == step {
		return errTest
	}
	return nil
}

type testInput func(ctx context.Context, st storage.Storage, bch <-chan struct{}) error

func inputRequest(req *neco.UpdateRequest) testInput {
	return func(ctx context.Context, st storage.Storage, bch <-chan struct{}) error {
		return st.PutRequest(ctx, *req, "hoge")
	}
}

func inputStatus(lrn int, status *neco.UpdateStatus, wait bool) testInput {
	return func(ctx context.Context, st storage.Storage, bch <-chan struct{}) error {
		if wait {
			<-bch
		}
		return st.PutStatus(ctx, lrn, *status)
	}
}

func inputClear() testInput {
	return func(ctx context.Context, st storage.Storage, bch <-chan struct{}) error {
		return st.ClearStatus(ctx)
	}
}

func testStatus(step int, cond neco.UpdateCondition) *neco.UpdateStatus {
	return &neco.UpdateStatus{
		Version: "1.0.0",
		Step:    step,
		Cond:    cond,
	}
}

var (
	testReq1 = &neco.UpdateRequest{
		Version: "1.1.0",
		Servers: []int{0, 1, 2},
	}
	testReq2 = &neco.UpdateRequest{
		Version: "1.1.0",
		Servers: []int{1, 2},
	}
	testReq = &neco.UpdateRequest{
		Version: "1.0.0",
		Servers: []int{0, 1},
		Stop:    false,
	}
	testReqStop = &neco.UpdateRequest{
		Version: "1.0.0",
		Servers: []int{0, 1},
		Stop:    true,
	}
)

func TestWorker(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  []testInput
		Op     *mockOp
		Expect *mockOp
		Error  bool
	}{
		{
			Name:   "update-neco",
			Input:  []testInput{inputRequest(testReq1)},
			Op:     newMock(false, 0),
			Expect: expect(true, 0, testReq1),
		},
		{
			Name:   "update-neco-fail",
			Input:  []testInput{inputRequest(testReq1)},
			Op:     newMock(true, 0),
			Expect: expect(false, 0, testReq1),
			Error:  true,
		},
		{
			Name:   "no-member",
			Input:  []testInput{inputRequest(testReq2)},
			Op:     newMock(false, 0),
			Expect: expect(false, 0, nil),
		},
		{
			Name: "previous-abort",
			Input: []testInput{
				inputStatus(0, testStatus(1, neco.CondAbort), false),
				inputStatus(1, testStatus(1, neco.CondRunning), false),
				inputRequest(testReq),
			},
			Op:     newMock(false, 0),
			Expect: expect(false, 0, nil),
		},
		{
			Name: "previous-abort-clear",
			Input: []testInput{
				inputStatus(0, testStatus(1, neco.CondAbort), false),
				inputStatus(1, testStatus(1, neco.CondRunning), false),
				inputRequest(testReqStop),
				inputClear(),
				inputRequest(testReq),
				inputStatus(1, testStatus(1, neco.CondRunning), false),
			},
			Op:     newMock(false, 0),
			Expect: expect(false, 1, testReq),
		},
		{
			Name: "previous-completed",
			Input: []testInput{
				inputStatus(0, testStatus(2, neco.CondComplete), false),
				inputStatus(1, testStatus(2, neco.CondComplete), false),
				inputRequest(testReq),
			},
			Op:     newMock(false, 0),
			Expect: expect(false, 0, nil),
		},
		{
			Name: "update-successful",
			Input: []testInput{
				inputRequest(testReq),
				inputStatus(1, testStatus(1, neco.CondRunning), false),
				inputStatus(3, testStatus(1, neco.CondRunning), false), // ignored
				inputStatus(1, testStatus(2, neco.CondRunning), true),
				inputStatus(1, testStatus(2, neco.CondComplete), false),
			},
			Op:     newMock(false, 0),
			Expect: expect(false, 2, testReq),
		},
		{
			Name: "update-successful-then-new-request",
			Input: []testInput{
				inputRequest(testReq),
				inputStatus(1, testStatus(1, neco.CondRunning), false),
				inputStatus(3, testStatus(1, neco.CondRunning), false), // ignored
				inputStatus(1, testStatus(2, neco.CondRunning), true),
				inputStatus(1, testStatus(2, neco.CondComplete), true),
				inputRequest(testReq1),
			},
			Op:     newMock(false, 0),
			Expect: expect(true, 2, testReq1),
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			ec := test.NewEtcdClient(t)
			defer ec.Close()
			_, err := ec.Put(context.Background(), "hoge", "")
			if err != nil {
				t.Fatal(err)
			}

			worker := NewWorker(ec, c.Op, "1.0.0", 0)
			st := storage.NewStorage(ec)

			var workerErr error
			env := well.NewEnvironment(context.Background())
			env.Go(func(ctx context.Context) error {
				ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
				defer cancel()
				err := worker.Run(ctx)
				if ctx.Err() != context.DeadlineExceeded {
					workerErr = err
				}
				return err
			})
			env.Go(func(ctx context.Context) error {
				for _, input := range c.Input {
					err := input(ctx, st, worker.bch)
					if err != nil {
						t.Log("input error!!", err)
						return err
					}
				}
				return nil
			})
			env.Stop()
			err = env.Wait()
			if err != nil {
				t.Log(err)
			}

			if !cmp.Equal(c.Op, c.Expect) {
				t.Errorf("unexpected result: expect=%+v, actual=%+v", c.Expect, c.Op)
			}

			if c.Error {
				if workerErr == nil {
					t.Error("error is expected")
				}
				return
			}

			if workerErr != nil {
				t.Error(workerErr)
			}
		})
	}
}
