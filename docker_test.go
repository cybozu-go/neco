package neco

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDockerRuntime(t *testing.T) {
	t.Skip("manually tested")

	rt, err := newDockerRuntime("")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	img, err := CurrentArtifacts.FindContainerImage("sabakan")
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Pull(ctx, img); err != nil {
		t.Fatal(err)
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := os.MkdirAll(filepath.Join(dir, "usr/local/bin"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := rt.Run(ctx, img, []Bind{
		{Name: "root", Source: dir, Dest: "/host", ReadOnly: false},
	}, []string{"/usr/local/sabakan/install-tools"}); err != nil {
		t.Error(err)
	}

	if err := rt.Exec(ctx, "ubuntu", false, []string{"touch", "/foo"}); err != nil {
		t.Error(err)
	}

	running, err := rt.IsRunning(img)
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Error(`not running`)
	}
}
