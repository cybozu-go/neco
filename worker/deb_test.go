package worker

import (
	"context"
	"net/http"
	"testing"

	"github.com/cybozu-go/neco"
)

func TestGetDebianVersion(t *testing.T) {
	t.Parallel()

	v, err := GetDebianVersion("bash")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(v)

	v, err = GetDebianVersion("no-such-package")
	if err != nil {
		t.Fatal(err)
	}
	if v != "" {
		t.Error("no-such-package version should be empty")
	}
}

func TestInstallDebianPackage(t *testing.T) {
	t.Skip()
	t.Parallel()

	pkg := &neco.DebianPackage{
		Name: "etcdpasswd", Owner: "cybozu-go", Repository: "etcdpasswd", Release: "v0.5",
	}

	err := InstallDebianPackage(context.Background(), http.DefaultClient, pkg)
	if err != nil {
		t.Fatal(err)
	}
}
