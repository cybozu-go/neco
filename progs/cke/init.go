package cke

import (
	"context"
	"path"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
	"github.com/hashicorp/vault/api"
)

// Init initialize cke for cluster
func Init(ctx context.Context, ec *clientv3.Client) error {
	return etcd.UserAdd(ctx, ec, "cke", neco.CKEPrefix)

	for _, ca := range cas {
		err = createCA(vc, ca)
		if err != nil {
			return err
		}
	}
	return nil
}

func createCA(vc *api.Client, ca ca) error {
	err := vc.Sys().Mount(ca.vaultPath, &api.MountInput{
		Type: "pki",
		Config: api.MountConfigInput{
			MaxLeaseTTL:     neco.TTL100Year,
			DefaultLeaseTTL: neco.TTL10Year,
		},
	})
	if err != nil {
		return err
	}

	secret, err := vc.Logical().Write(path.Join(ca.vaultPath, "/root/generate/internal"), map[string]interface{}{
		"common_name": ca.commonName,
		"ttl":         neco.TTL100Year,
		"format":      "pem",
	})
	if err != nil {
		return err
	}
	_ = secret
	//        with tempfile.NamedTemporaryFile(mode='w') as f:
	//            f.write(secret['data']['certificate'])
	//            f.flush()
	return nil
}
