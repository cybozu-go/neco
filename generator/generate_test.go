package generator

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cybozu-go/neco"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/name"
)

func TestGetLatestImage(t *testing.T) {
	t.Parallel()

	type args struct {
		repo            name.Repository
		release         bool
		ignoredVersions []string
	}

	var repoName = "cybozu/ubuntu"
	var bindAddr = "127.0.0.1:8000"
	testRepository, err := name.NewRepository(fmt.Sprintf("%s/%s", bindAddr, repoName), name.WeakValidation)
	if err != nil {
		t.Fatalf("failed to fetch test repository: %v", err)
	}

	tests := []struct {
		name     string
		args     args
		tagsList []byte
		wantErr  bool
		img      *neco.ContainerImage
	}{
		{
			name: "should get latest image",
			args: args{
				repo:    testRepository,
				release: false,
			},
			tagsList: []byte(`{"tags":["0.1.0","0.2.0", "1.0.0"]}`),
			img: &neco.ContainerImage{
				Name:       imageName(testRepository),
				Repository: testRepository.String(),
				Tag:        "1.0.0",
				Private:    false,
			},
		},
		{
			name: "should ignore specific tags",
			args: args{
				repo:            testRepository,
				release:         false,
				ignoredVersions: []string{"1.0.0"},
			},
			tagsList: []byte(`{"tags":["0.1.0","0.2.0", "1.0.0"]}`),
			img: &neco.ContainerImage{
				Name:       imageName(testRepository),
				Repository: testRepository.String(),
				Tag:        "0.2.0",
				Private:    false,
			},
		},
		{
			name: "can set ignored versions that is no-release version",
			args: args{
				repo:            testRepository,
				release:         false,
				ignoredVersions: []string{"100.0.0"},
			},
			tagsList: []byte(`{"tags":["0.1.0","0.2.0", "1.0.0"]}`),
			img: &neco.ContainerImage{
				Name:       imageName(testRepository),
				Repository: testRepository.String(),
				Tag:        "1.0.0",
				Private:    false,
			},
		},
		{
			name: "should get latest image with release",
			args: args{
				repo:    testRepository,
				release: true,
			},
			tagsList: []byte(`{"tags":["0.1.0","0.2.0", "1.0.0"]}`),
			img: &neco.ContainerImage{
				Name:       imageName(testRepository),
				Repository: testRepository.String(),
				Tag:        "1.0.0",
				Private:    false,
			},
		},
		{
			name: "should return an error if only branch tags are available",
			args: args{
				repo:    testRepository,
				release: true,
			},
			tagsList: []byte(`{"tags":["0.1"]}`),
			wantErr:  true,
			img:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tagsPath := fmt.Sprintf("/v2/%s/tags/list", repoName)
			l, err := net.Listen("tcp", bindAddr)
			if err != nil {
				t.Fatalf("failed to listen: %v", err)
			}
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/":
					w.WriteHeader(http.StatusOK)
				case tagsPath:
					if r.Method != http.MethodGet {
						t.Errorf("unable to handle method called: %v", r.Method)
					}
					w.Write(test.tagsList)
				default:
					t.Fatalf("unable to handle path called: %v", r.URL.Path)
				}
			}))
			server.Listener.Close()
			server.Listener = l
			server.Start()
			defer server.Close()

			img, err := getLatestImage(context.Background(), test.args.repo, test.args.ignoredVersions)
			if err != nil && !test.wantErr {
				t.Errorf("failed to getLatestImage: %v", err)
			}

			if diff := cmp.Diff(img, test.img); diff != "" {
				t.Errorf("got: %v, want: %v", img, test.img)
			}
		})
	}
}
