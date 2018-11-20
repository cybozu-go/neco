package worker

import (
	"context"
	"net/http"
	"testing"

	"github.com/cybozu-go/neco"
)

func TestInstallDebianPackage(t *testing.T) {
	t.Skip()
	t.Parallel()

	pkg := &neco.DebianPackage{
		Name: "etcdpasswd", Owner: "cybozu-go", Repository: "etcdpasswd", Release: "v0.5",
	}

	err := InstallDebianPackage(context.Background(), http.DefaultClient, pkg, true)
	if err != nil {
		t.Fatal(err)
	}
}
