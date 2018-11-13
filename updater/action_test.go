package updater

import (
	"context"
	"testing"
	"time"

	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/updater/mock"
)

func TestNextAction(t *testing.T) {
	ctx := context.Background()
	pkg := mock.PackageManager{}
	timeout := time.Hour
	tests := []struct {
		name    string
		ss      *storage.Snapshot
		want    Action
		wantErr bool
	}{
		{
			name:    "still release-checker is not running",
			ss:      &storage.Snapshot{Latest: ""},
			want:    ActionWaitInfo,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextAction(ctx, tt.ss, pkg, timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("NextAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NextAction() = %v, want %v", got, tt.want)
			}
		})
	}
}
