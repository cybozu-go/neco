package cke

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/etcdutil"
)

const (
	defaultEtcdPrefix = "/cke/"
)

// NewEtcdConfig creates Config with default prefix.
func NewEtcdConfig() *etcdutil.Config {
	return etcdutil.NewConfig(defaultEtcdPrefix)
}

// AddUserRole create etcd user and role.
func AddUserRole(ctx context.Context, cli *clientv3.Client, name, prefix string) error {
	r := make([]byte, 32)
	_, err := rand.Read(r)
	if err != nil {
		return err
	}

	_, err = cli.UserAdd(ctx, name, hex.EncodeToString(r))
	if err != nil {
		return err
	}

	if prefix == "" {
		return nil
	}

	_, err = cli.RoleAdd(ctx, name)
	if err != nil {
		return err
	}

	_, err = cli.RoleGrantPermission(ctx, name, prefix, clientv3.GetPrefixRangeEnd(prefix), clientv3.PermissionType(clientv3.PermReadWrite))
	if err != nil {
		return err
	}

	_, err = cli.UserGrantRole(ctx, name, name)
	if err != nil {
		return err
	}

	return nil
}

// GetUserRoles get roles of target user.
func GetUserRoles(ctx context.Context, cli *clientv3.Client, user string) ([]string, error) {
	resp, err := cli.UserGet(ctx, user)
	return resp.Roles, err
}
