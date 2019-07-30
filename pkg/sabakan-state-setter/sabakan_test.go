package sss

import (
	"testing"
	"time"
)

func TestNeedUpdateState(t *testing.T) {
	now := time.Date(2019, 7, 20, 0, 0, 0, 0, time.Local)

	type fields struct {
		stateCandidate               string
		stateCandidateFirstDetection time.Time
		machineType                  *machineType
	}
	type args struct {
		newState string
		now      time.Time
	}
	defaultMachineType := &machineType{GracePeriod: duration{time.Minute}}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "do not need update: no much time yet",
			fields: fields{"unhealthy", now.Add(-time.Second), defaultMachineType},
			args:   args{"unhealthy", now},
			want:   false,
		},
		{
			name:   "do not need update: state is changed",
			fields: fields{"unhealthy", now.Add(-time.Hour), defaultMachineType},
			args:   args{"unreachable", now},
			want:   false,
		},
		{
			name:   "need update: 2 min left unhealthy",
			fields: fields{"unhealthy", now.Add(-2 * time.Minute), defaultMachineType},
			args:   args{"unhealthy", now},
			want:   true,
		},
		{
			name:   "need update: state is not problematic",
			fields: fields{"healthy", now.Add(-time.Hour), defaultMachineType},
			args:   args{"healthy", now},
			want:   true,
		},
		{
			name:   "no state transition",
			fields: fields{"unhealthy", now.Add(-time.Hour), defaultMachineType},
			args:   args{noStateTransition, now},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &machineStateSource{
				stateCandidate:               tt.fields.stateCandidate,
				stateCandidateFirstDetection: tt.fields.stateCandidateFirstDetection,
				machineType:                  tt.fields.machineType,
			}

			if got := ms.needUpdateState(tt.args.newState, tt.args.now); got != tt.want {
				t.Errorf("MachineStateSource.needUpdateState() = %v, want %v", got, tt.want)
			}
		})
	}
}
