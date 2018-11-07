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
	FailNecoUpdate  bool
	FailEtcdUpdate  bool
	FailVaultUpdate bool

	NecoUpdated  bool
	EtcdUpdated  bool
	VaultUpdated bool
	Req          *neco.UpdateRequest
}

func newMock(failNeco, failEtcd, failVault bool) *mockOp {
	return &mockOp{
		FailNecoUpdate:  failNeco,
		FailEtcdUpdate:  failEtcd,
		FailVaultUpdate: failVault,
	}
}

func expect(necoUpdated, etcdUpdated, vaultUpdated bool, req *neco.UpdateRequest) *mockOp {
	return &mockOp{
		NecoUpdated:  necoUpdated,
		EtcdUpdated:  etcdUpdated,
		VaultUpdated: vaultUpdated,
		Req:          req,
	}
}

func (op *mockOp) Equal(expected *mockOp) bool {
	if op.NecoUpdated != expected.NecoUpdated {
		return false
	}
	if op.EtcdUpdated != expected.EtcdUpdated {
		return false
	}
	if op.VaultUpdated != expected.VaultUpdated {
		return false
	}
	return cmp.Equal(op.Req, expected.Req)
}

func (op *mockOp) UpdateEtcd(ctx context.Context, req *neco.UpdateRequest) error {
	op.Req = req
	if op.FailEtcdUpdate {
		return errTest
	}
	op.EtcdUpdated = true
	return nil
}

func (op *mockOp) UpdateVault(ctx context.Context, req *neco.UpdateRequest) error {
	op.Req = req
	if op.FailVaultUpdate {
		return errTest
	}
	op.VaultUpdated = true
	return nil
}

func (op *mockOp) UpdateNeco(ctx context.Context, req *neco.UpdateRequest) error {
	op.Req = req
	if op.FailNecoUpdate {
		return errTest
	}
	op.NecoUpdated = true
	return nil
}

type testInput struct {
	req    *neco.UpdateRequest
	lrn    int
	status *neco.UpdateStatus
}

var (
	testReq1 = &neco.UpdateRequest{
		Version: "1.1.0",
		Servers: []int{0, 1, 2},
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
			Name: "update-neco",
			Input: []testInput{
				{
					req: testReq1,
				},
			},
			Op:     newMock(false, false, false),
			Expect: expect(true, false, false, testReq1),
		},
	}

	for _, c := range testCases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			ec := test.NewEtcdClient(t)
			_, err := ec.Put(context.Background(), "hoge", "")
			if err != nil {
				t.Fatal(err)
			}

			worker := NewWorker(ec, c.Op, "1.0.0", 0)

			var workerErr error
			env := well.NewEnvironment(context.Background())
			env.Go(func(ctx context.Context) error {
				workerErr = worker.Run(ctx)
				return workerErr
			})
			env.Go(func(ctx context.Context) error {
				st := storage.NewStorage(ec)
				for _, input := range c.Input {
					if input.req != nil {
						err := st.PutRequest(ctx, *input.req, "hoge")
						if err != nil {
							return err
						}
						continue
					}
					err := st.PutStatus(ctx, input.lrn, *input.status)
					if err != nil {
						return err
					}
				}
				time.Sleep(500 * time.Millisecond)
				return errTest
			})
			env.Stop()
			env.Wait()

			if c.Error {
				if workerErr == nil {
					t.Error("error is expected")
				}
				return
			}

			if workerErr != nil {
				t.Error(workerErr)
			}
			if !cmp.Equal(c.Op, c.Expect) {
				t.Errorf("unexpected result: expect=%+v, actual=%+v", c.Expect, c.Op)
			}
		})
	}
}
