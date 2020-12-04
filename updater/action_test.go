package updater

import (
	"testing"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

func TestNextAction(t *testing.T) {
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
			ss:   &storage.Snapshot{Latest: "", Request: req},
			want: ActionWaitInfo,
		},
		{
			name: "recover",
			ss:   &storage.Snapshot{Latest: "1.0.0"},
			want: ActionNewVersion,
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
			got, err := NextAction(tt.ss, timeout)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("NextAction() = %v, want %v", got, tt.want)
			}
		})
	}
}
