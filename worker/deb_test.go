package worker

import (
	"reflect"
	"testing"

	"github.com/google/go-github/v35/github"
)

func TestFindDebAsset(t *testing.T) {
	type args struct {
		assets []*github.ReleaseAsset
		name   string
	}
	tests := []struct {
		name string
		args args
		want *github.ReleaseAsset
	}{
		{
			"valid",
			args{
				assets: []*github.ReleaseAsset{{Name: strToPointer("etcdpasswd_aaa.deb")}},
				name:   "etcdpasswd",
			},
			&github.ReleaseAsset{Name: strToPointer("etcdpasswd_aaa.deb")},
		},
		{
			"invalid ext",
			args{
				assets: []*github.ReleaseAsset{{Name: strToPointer("etcdpasswd_aaa.zip")}},
				name:   "etcdpasswd",
			},
			nil,
		},
		{
			"name not matched",
			args{
				assets: []*github.ReleaseAsset{{Name: strToPointer("etcdpasswd_aaa.deb")}},
				name:   "neco",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findDebAsset(tt.args.assets, tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findDebAsset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func strToPointer(str string) *string {
	return &str
}
