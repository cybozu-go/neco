package setup

import (
	"context"
	"os/exec"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/hashicorp/vault/api"
)

const vaultPath = "/usr/local/bin/vault"

func createCA(ctx context.Context, vault *api.Client) ([]*api.Secret, error) {
	for _, ca := range []string{neco.CAServer, neco.CAEtcdPeer, neco.CAEtcdClient} {
		err := vault.Sys().Mount(ca, &api.MountInput{
			Type: "pki",
			Config: api.MountConfigInput{
				MaxLeaseTTL:     neco.TTL100Year,
				DefaultLeaseTTL: neco.TTL10Year,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	serverPem, err := vault.Logical().Write(neco.CAServer+"/root/generate/exported", map[string]interface{}{
		"ttl":         neco.TTL100Year,
		"common_name": "server CA",
		"format":      "pem_bundle",
	})
	if err != nil {
		return nil, err
	}
	peerPem, err := vault.Logical().Write(neco.CAEtcdPeer+"/root/generate/exported", map[string]interface{}{
		"ttl":         neco.TTL100Year,
		"common_name": "boot etcd peer CA",
		"format":      "pem_bundle",
	})
	if err != nil {
		return nil, err
	}
	clientPem, err := vault.Logical().Write(neco.CAEtcdClient+"/root/generate/exported", map[string]interface{}{
		"ttl":         neco.TTL100Year,
		"common_name": "boot etcd client CA",
		"format":      "pem_bundle",
	})
	if err != nil {
		return nil, err
	}

	// bootstrap certificates should expire really soon (one hour).
	_, err = vault.Logical().Write(neco.CAServer+"/roles/system", map[string]interface{}{
		"ttl":            "1h",
		"max_ttl":        "1h",
		"client_flag":    false,
		"allow_any_name": true,
	})
	if err != nil {
		return nil, err
	}

	_, err = vault.Logical().Write(neco.CAEtcdPeer+"/roles/system", map[string]interface{}{
		"ttl":            "1h",
		"max_ttl":        "1h",
		"allow_any_name": true,
	})
	if err != nil {
		return nil, err
	}

	_, err = vault.Logical().Write(neco.CAEtcdClient+"/roles/system", map[string]interface{}{
		"ttl":            "1h",
		"max_ttl":        "1h",
		"server_flag":    false,
		"allow_any_name": true,
	})
	if err != nil {
		return nil, err
	}

	vault.Logical().Write("secret/bootstrap", map[string]interface{}{
		"ready": "go",
	})
	return []*api.Secret{serverPem, peerPem, clientPem}, nil
}

func prepareCA(ctx context.Context, isLeader bool, mylrn int, lrns []int) ([]*api.Secret, error) {
	err := neco.RunContainer(ctx, "vault",
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/vault/install-tools"})
	if err != nil {
		return nil, err
	}

	cfg := api.DefaultConfig()
	cfg.Address = "http://127.0.0.1:8200"
	vault, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	vault.SetToken("cybozu")

	if isLeader {
		tmpCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		cmd := exec.CommandContext(tmpCtx, vaultPath, "server", "-dev",
			"-dev-listen-address=0.0.0.0:8200", "-dev-root-token-id=cybozu")
		err = cmd.Start()
		if err != nil {
			return nil, err
		}

		time.Sleep(1 * time.Second)

		return createCA(ctx, vault)
	}

	// TODO
	return nil, nil
}
