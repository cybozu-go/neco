package cmd

import (
	"context"
	"testing"

	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/storage/test"
	"github.com/google/go-cmp/cmp"
)

func TestGetAuthServers(t *testing.T) {
	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	st := storage.NewStorage(etcd)
	ctx := context.Background()

	expect := []string{
		"10.10.10.1",
		"10.10.10.2",
	}

	err := st.PutTeleportAuthServers(ctx, expect)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := getAuthServers(st)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(expect, actual) {
		t.Errorf("does not match: expect %v, actual %v", expect, actual)
	}
}

func TestGenerateConfig(t *testing.T) {
	expect := `teleport:
  auth_token: dummy-auth-token
  auth_servers: ["10.10.10.1","10.10.10.2"]
`

	base := `teleport:
  auth_token: %AUTH_TOKEN%
  auth_servers: %AUTH_SERVERS%
`
	token := "dummy-auth-token"
	authServers := []string{
		"10.10.10.1",
		"10.10.10.2",
	}
	actual, err := generateConfig([]byte(token), authServers, []byte(base))
	if err != nil {
		t.Fatal(err)
	}

	if string(actual) != expect {
		t.Errorf("does not match: expect %s, actual %s", expect, string(actual))
	}
}
