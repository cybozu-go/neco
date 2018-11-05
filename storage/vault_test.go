package storage

import (
	"context"
	"testing"
)

func testVaultUnsealKey(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetVaultUnsealKey(ctx)
	if err != ErrNotFound {
		t.Error("unexpected error", err)
	}

	err = st.PutVaultUnsealKey(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := st.GetVaultUnsealKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resp != "key" {
		t.Error("wrong vault unseal key")
	}

	err = st.DeleteVaultUnsealKey(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = st.GetVaultUnsealKey(ctx)
	if err != ErrNotFound {
		t.Error("unexpected error", err)
	}
}

func testVaultRootToken(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetVaultRootToken(ctx)
	if err != ErrNotFound {
		t.Error("unexpected error", err)
	}

	err = st.PutVaultRootToken(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := st.GetVaultRootToken(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resp != "root" {
		t.Error("wrong vault root token")
	}

	err = st.DeleteVaultRootToken(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.GetVaultRootToken(ctx)
	if err != ErrNotFound {
		t.Error("unexpected error", err)
	}
}

func TestVault(t *testing.T) {
	t.Run("unseal-key", testVaultUnsealKey)
	t.Run("root-token", testVaultRootToken)
}
