package cke

import (
	"context"
	"path"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
	"github.com/hashicorp/vault/api"
)

const (
	caServer     = "cke/ca-server"
	caEtcdPeer   = "cke/ca-etcd-peer"
	caEtcdClient = "cke/ca-etcd-client"
	caKubernetes = "cke/ca-kubernetes"
)

// Init initialize cke for cluster
func Init(ctx context.Context, ec *clientv3.Client) error {
	err := etcd.UserAdd(ctx, ec, "cke", neco.CKEPrefix)
	if err != nil {
		return err
	}

	mylrn, err := neco.MyLRN()
	if err != nil {
		return err
	}
	vc, err := neco.VaultClient(mylrn)
	if err != nil {
		return err
	}

	err = vc.Sys().PutPolicy("cke", ckePolicy)
	if err != nil {
		return err
	}

	_, err = vc.Logical().Write("auth/approle/role/cke", map[string]interface{}{
		"policies": "cke",
		"period":   "1h",
	})

	//	secret, err := vc.Logical().Read("auth/approle/role/cke/role-id")
	//	if err != nil {
	//		return err
	//	}
	//	roleID = secret.Data["role_id"].(string)
	//
	//	secret, err := vc.Logical().Write("auth/approle/role/cke/secret-id", map[string]interface{}{})
	//	if err != nil {
	//		return err
	//	}
	//	secretID = secret.Data["secret_id"].(string)
	//
	//	//    data = {'endpoint': 'https://localhost:8200',
	//	//            'role-id': role_id,
	//	//            'secret-id': secret_id}
	//	//    ckecli('vault', 'config', '-', input=json.dumps(data).encode('utf-8'))
	//
	//	secret, err := vc.Logical().Write("auth/approle/login", map[string]interface{}{
	//		"role_id":   roleID,
	//		"secret_id": secretID,
	//	})
	//	if err != nil {
	//		return err
	//	}
	//	approleToken = secret.Auth.ClientToken
	//
	//	err = createCA(caServer, "server CA", "server")
	//	if err != nil {
	//		return err
	//	}
	//	err = createCA(caEtcdPeer, "etcd peer CA", "etcd-peer")
	//	if err != nil {
	//		return err
	//	}
	//	err = createCA(caEtcdClient, "etcd client CA", "etc-client")
	//	if err != nil {
	//		return err
	//	}
	//	return createCA(caKubernetes, "kubernetes CA", "kubernetes")
	return nil
}

func createCA(vc *api.Client, ca, cn, key string) error {
	err := vc.Sys().Mount(ca, &api.MountInput{
		Type: "pki",
		Config: api.MountConfigInput{
			MaxLeaseTTL:     neco.TTL100Year,
			DefaultLeaseTTL: neco.TTL10Year,
		},
	})
	if err != nil {
		return err
	}

	secret, err := vc.Logical().Write(path.Join(ca, "/root/generate/internal"), map[string]interface{}{
		"common_name": cn,
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
