package storage

import (
	"context"
	"testing"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage/test"
)

func testEnvConfig(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	env, err := st.GetEnvConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if env != neco.NoneEnv {
		t.Error(`env != neco.NoneEnv`, env)
	}

	err = st.PutEnvConfig(ctx, neco.StagingEnv)
	if err != nil {
		t.Fatal(err)
	}

	env, err = st.GetEnvConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if env != neco.StagingEnv {
		t.Error(`env != neco.StagingEnv"`, env)
	}
}

func testSlackNotification(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetSlackNotification(ctx)
	if err != ErrNotFound {
		t.Error("notification config should not be found")
	}

	err = st.PutSlackNotification(ctx, "http://slack.com/aaa")
	if err != nil {
		t.Fatal(err)
	}

	url, err := st.GetSlackNotification(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if url != "http://slack.com/aaa" {
		t.Error(`nc.Slack != "http://slack.com/aaa"`, url)
	}
}

func testProxyConfig(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetProxyConfig(ctx)
	if err != ErrNotFound {
		t.Error("proxy config should not be found")
	}

	err = st.PutProxyConfig(ctx, "http://squid.example.com:3128")
	if err != nil {
		t.Fatal(err)
	}

	proxy, err := st.GetProxyConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if proxy != "http://squid.example.com:3128" {
		t.Error(`proxy != "http://squid.example.com:3128"`, proxy)
	}
}

func testGhcr(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetGhcrUsername(ctx)
	if err != ErrNotFound {
		t.Error("ghcr username should not be found")
	}
	_, err = st.GetGhcrPassword(ctx)
	if err != ErrNotFound {
		t.Error("ghcr password should not be found")
	}

	err = st.PutGhcrUsername(ctx, "fooUser")
	if err != nil {
		t.Fatal(err)
	}
	err = st.PutGhcrPassword(ctx, "fooPassword")
	if err != nil {
		t.Fatal(err)
	}

	username, err := st.GetGhcrUsername(ctx)
	if err != nil {
		t.Fatal(err)
	}
	password, err := st.GetGhcrPassword(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if username != "fooUser" {
		t.Error(`username != "fooUser"`, username)
	}
	if password != "fooPassword" {
		t.Error(`password != "fooPassword"`, password)
	}
}

func testCheckUpdateIntervalConfig(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	d, err := st.GetCheckUpdateInterval(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d != DefaultCheckUpdateInterval {
		t.Error(`d != DefaultCheckUpdateInterval`, d)
	}

	err = st.PutCheckUpdateInterval(ctx, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	d, err = st.GetCheckUpdateInterval(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d != 10*time.Minute {
		t.Error(`d != 10*time.Minute`, d)
	}
}

func testWorkerTimeout(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	d, err := st.GetWorkerTimeout(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d != DefaultWorkerTimeout {
		t.Error(`d != DefaultWorkerTimeout`, d)
	}

	err = st.PutWorkerTimeout(ctx, 60*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	d, err = st.GetWorkerTimeout(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d != 60*time.Minute {
		t.Error(`d != 10*time.Minute`, d)
	}
}

func TestConfig(t *testing.T) {
	t.Run("EnvConfig", testEnvConfig)
	t.Run("SlackNotification", testSlackNotification)
	t.Run("ProxyConfig", testProxyConfig)
	t.Run("Ghcr", testGhcr)
	t.Run("CheckUpdateIntervalConfig", testCheckUpdateIntervalConfig)
	t.Run("WorkerTimeout", testWorkerTimeout)
}
