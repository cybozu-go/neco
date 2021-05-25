package etcd

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"go.etcd.io/etcd/clientv3"
)

func rpgen() (string, error) {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

// UserAdd adds a user to etcd.
//
// If prefix is empty, the user will have "root" role.
//
// Otherwise, the user will have a role whose name is the same as the user,
// and grants read-write access under prefix.
func UserAdd(ctx context.Context, ec *clientv3.Client, name, prefix string) error {
	if name == "root" {
		_, err := ec.RoleAdd(ctx, "root")
		if err != nil {
			return err
		}
	}

	rp, err := rpgen()
	if err != nil {
		return err
	}
	_, err = ec.UserAdd(ctx, name, rp)
	if err != nil {
		return err
	}

	if prefix == "" {
		_, err = ec.UserGrantRole(ctx, name, "root")
		return err
	}

	_, err = ec.RoleAdd(ctx, name)
	if err != nil {
		return err
	}

	_, err = ec.RoleGrantPermission(ctx, name,
		prefix, clientv3.GetPrefixRangeEnd(prefix),
		clientv3.PermissionType(clientv3.PermReadWrite))
	if err != nil {
		return err
	}

	_, err = ec.UserGrantRole(ctx, name, name)
	return err
}
