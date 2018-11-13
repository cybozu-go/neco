package updater

import (
	"context"
	"testing"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

type testPackageManager struct {
	version string
}

func (m testPackageManager) GetVersion(ctx context.Context, name string) (string, error) {
	return m.version, nil
}

func TestNextAction(t *testing.T) {
	ctx := context.Background()
	timeout := time.Hour
	req := &neco.UpdateRequest{
		Version:   "1.0.0",
		Servers:   []int{0, 1},
		StartedAt: time.Now(),
	}
	oldReq := *req
	oldReq.StartedAt = time.Now().Add(-2 * timeout)
	statuses := map[int]*neco.UpdateStatus{
		0: {
			Version: "1.0.0",
			Step:    3,
			Cond:    neco.CondComplete,
		},
		1: {
			Version: "1.0.0",
			Step:    3,
			Cond:    neco.CondComplete,
		},
	}

	tests := []struct {
		name string
		ss   *storage.Snapshot
		want Action
	}{
		{
			name: "wait-latest",
			ss:   &storage.Snapshot{Latest: ""},
			want: ActionWaitInfo,
		},
		{
			name: "new-release",
			ss:   &storage.Snapshot{Latest: "1.1.1"},
			want: ActionNewVersion,
		},
		{
			name: "no-new-release",
			ss:   &storage.Snapshot{Latest: "1.0.0"},
			want: ActionWaitInfo,
		},
		{
			name: "stopped",
			ss: &storage.Snapshot{
				Latest:  "1.0.0",
				Request: &neco.UpdateRequest{Stop: true},
			},
			want: ActionWaitClear,
		},
		{
			name: "aborted",
			ss: &storage.Snapshot{
				Latest:  "1.0.0",
				Request: req,
				Statuses: map[int]*neco.UpdateStatus{
					0: {
						Version: "1.0.0",
						Cond:    neco.CondAbort,
					},
				},
			},
			want: ActionStop,
		},
		{
			name: "timeout",
			ss: &storage.Snapshot{
				Latest:  "1.0.0",
				Request: &oldReq,
			},
			want: ActionStop,
		},
		{
			name: "not-completed",
			ss: &storage.Snapshot{
				Latest:  "1.0.0",
				Request: req,
			},
			want: ActionWaitWorkers,
		},
		{
			name: "reconfigure",
			ss: &storage.Snapshot{
				Latest:   "1.0.0",
				Request:  req,
				Statuses: statuses,
				Servers:  []int{0, 1, 2},
			},
			want: ActionReconfigure,
		},
		{
			name: "update",
			ss: &storage.Snapshot{
				Latest:   "1.1.0",
				Request:  req,
				Statuses: statuses,
				Servers:  []int{0, 1},
			},
			want: ActionNewVersion,
		},
		{
			name: "completed",
			ss: &storage.Snapshot{
				Latest:   "1.0.0",
				Request:  req,
				Statuses: statuses,
				Servers:  []int{0, 1},
			},
			want: ActionWaitInfo,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pkg := testPackageManager{"1.0.0"}
			got, err := NextAction(ctx, tt.ss, pkg, timeout)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("NextAction() = %v, want %v", got, tt.want)
			}
		})
	}
}
