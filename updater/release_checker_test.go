package updater

import (
	"context"
	"testing"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/storage/test"
	"github.com/cybozu-go/neco/updater/mock"
)

func testUpdate(t *testing.T) {
	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := storage.NewStorage(etcd)

	checker := NewReleaseChecker(st,
		mock.GitHub{Release: "1.0.0", PreRelease: "0.1.0"},
		mock.PackageManager{Versions: map[string]string{"neco": "0.0.0"}})

	err := checker.update(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version := checker.GetLatest(); version != "" {
		t.Error(`version != "":`, version)
	}

	err = st.PutEnvConfig(ctx, neco.StagingEnv)
	if err != nil {
		t.Fatal(err)
	}
	err = checker.update(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version := checker.GetLatest(); version != "0.1.0" {
		t.Error(`version != "0.1.0":`, version)
	}

	err = st.PutEnvConfig(ctx, neco.ProdEnv)
	if err != nil {
		t.Fatal(err)
	}
	err = checker.update(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version := checker.GetLatest(); version != "1.0.0" {
		t.Error(`version != "1.0.0":`, version)
	}
}

func TestReleaseChecker(t *testing.T) {
	t.Run("Update", testUpdate)
}
