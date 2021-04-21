package updater

import (
	"reflect"
	"testing"
	"time"
)

func TestNewNecoRelease(t *testing.T) {
	type args struct {
		tag string
	}
	tests := []struct {
		name    string
		args    args
		want    *necoRelease
		wantErr bool
	}{
		{
			name:    "should succeed to create neco release for prod",
			args:    args{"release-2021.04.19-13965"},
			want:    &necoRelease{"release", time.Date(2021, time.April, 19, 0, 0, 0, 0, time.UTC), 13965},
			wantErr: false,
		},
		{
			name:    "should fail to create neco release with invalid tag format",
			args:    args{"release-2021.04.19"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "should fail to create neco release with invalid time format",
			args:    args{"release-20210419-13965"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "should fail to create neco release with invalid release version",
			args:    args{"release-2021.04.19-abc"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newNecoRelease(tt.args.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("newNecoRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newNecoRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNecoReleaseIsNewerThan(t *testing.T) {
	type fields struct {
		prefix  string
		date    time.Time
		version int
	}
	type args struct {
		target *necoRelease
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "should be newer than given neco release argument on version and date",
			fields: fields{"release", time.Date(2021, time.April, 20, 0, 0, 0, 0, time.UTC), 13966},
			args:   args{&necoRelease{"release", time.Date(2021, time.April, 19, 0, 0, 0, 0, time.UTC), 13965}},
			want:   true,
		},
		{
			name:   "should be newer than given neco release argument on version",
			fields: fields{"release", time.Date(2021, time.April, 19, 0, 0, 0, 0, time.UTC), 13966},
			args:   args{&necoRelease{"release", time.Date(2021, time.April, 19, 0, 0, 0, 0, time.UTC), 13965}},
			want:   true,
		},
		{
			name:   "should be older than given neco release argument on version and date",
			fields: fields{"release", time.Date(2021, time.April, 18, 0, 0, 0, 0, time.UTC), 13964},
			args:   args{&necoRelease{"release", time.Date(2021, time.April, 19, 0, 0, 0, 0, time.UTC), 13965}},
			want:   false,
		},
		{
			name:   "should be older than given neco release argument on date",
			fields: fields{"release", time.Date(2021, time.April, 19, 0, 0, 0, 0, time.UTC), 13964},
			args:   args{&necoRelease{"release", time.Date(2021, time.April, 19, 0, 0, 0, 0, time.UTC), 13965}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := necoRelease{
				prefix:  tt.fields.prefix,
				date:    tt.fields.date,
				version: tt.fields.version,
			}
			got := r.isNewerThan(tt.args.target)
			if got != tt.want {
				t.Errorf("necoRelease.isNewerThan() = %v, want %v", got, tt.want)
			}
		})
	}
}
