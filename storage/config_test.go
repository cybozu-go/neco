package storage

import (
	"context"
	"testing"
	"time"

	"github.com/cybozu-go/neco"
)

func testEnvConfig(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetEnvConfig(ctx)
	if err != ErrNotFound {
		t.Error("env config should not be found")
	}

	err = st.PutEnvConfig(ctx, "http://squid.example.com:3128")
	if err != nil {
		t.Fatal(err)
	}

	proxy, err := st.GetEnvConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if proxy != "http://squid.example.com:3128" {
		t.Error(`proxy != "http://squid.example.com:3128"`, proxy)
	}
}

func testSlackNotification(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
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

	etcd := newEtcdClient(t)
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

func testCheckUpdateIntervalConfig(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	d, err := st.GetCheckUpdateInterval(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d != neco.DefaultCheckUpdateInterval {
		t.Error(`d != neco.DefaultCheckUpdateInterval`, d)
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

func TestConfig(t *testing.T) {
	t.Run("EnvConfig", testEnvConfig)
	t.Run("SlackNotification", testSlackNotification)
	t.Run("ProxyConfig", testProxyConfig)
	t.Run("CheckUpdateIntervalConfig", testCheckUpdateIntervalConfig)
}
