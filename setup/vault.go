package setup

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	"github.com/hashicorp/vault/api"
)

const vaultPath = "/usr/local/bin/vault"

func dumpCertFiles(secret *api.Secret, caFile, certFile, keyFile string) error {
	err := ioutil.WriteFile(certFile, secret.Data["certificate"].([]byte), 0644)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(keyFile, secret.Data["private_key"].([]byte), 0644)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(caFile, secret.Data["issuing_ca"].([]byte), 0644)
}

func setupLocalCerts(ctx context.Context, vault *api.Client, lrn int) error {
	for {
		_, err := vault.Logical().Read("secret/bootstrap")
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(1 * time.Second):
		}
	}

	log.Info("prepare: begin", nil)

	myname, err := os.Hostname()
	if err != nil {
		return err
	}
	myip := neco.BootNode0IP(lrn)

	secret, err := vault.Logical().Write(neco.CAServer+"/issue/system", map[string]interface{}{
		"common_name": myname,
		"alt_names":   "localhost",
		"ip_sans":     []string{"127.0.0.1", myip.String()},
	})
	if err != nil {
		return err
	}
	err = dumpCertFiles(secret, neco.ServerCAFile, neco.ServerCertFile, neco.ServerKeyFile)
	if err != nil {
		return err
	}

	err = well.CommandContext(ctx, "update-ca-certificates").Run()
	if err != nil {
		return err
	}

	bip, err := bastionIP()
	if err != nil {
		return err
	}

	// issue client certificate for etcd peer
	secret, err = vault.Logical().Write(neco.CAEtcdPeer+"/issue/system", map[string]interface{}{
		"common_name":          myname,
		"ip_sans":              []string{myip.String(), bip.String()},
		"exclude_cn_from_sans": true,
	})
	err = dumpCertFiles(secret, neco.EtcdPeerCAFile, neco.EtcdPeerCertFile, neco.EtcdPeerKeyFile)
	if err != nil {
		return err
	}

	// issue client certificate for vault to connect etcd
	secret, err = vault.Logical().Write(neco.CAEtcdClient+"/issue/system", map[string]interface{}{
		"common_name":          "vault",
		"exclude_cn_from_sans": true,
	})
	err = dumpCertFiles(secret, neco.EtcdClientCAFile, neco.VaultCertFile, neco.VaultKeyFile)
	if err != nil {
		return err
	}

	_, err = vault.Logical().Write(fmt.Sprintf("secret/bootstrap_done/%d", lrn), map[string]interface{}{"done": 1})
	if err != nil {
		return err
	}
	log.Info("prepare: end", nil)

	return nil
}

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
	cfg.Address = fmt.Sprintf("http://%s:8200", neco.BootNode0IP(lrns[0]).String())
	vault, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	vault.SetToken("cybozu")

	if isLeader {
		err := setupLocalCerts(ctx, vault, mylrn)
		if err != nil {
			return nil, err
		}

		tmpCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		cmd := exec.CommandContext(tmpCtx, vaultPath, "server", "-dev",
			"-dev-listen-address=0.0.0.0:8200", "-dev-root-token-id=cybozu")
		err = cmd.Start()
		if err != nil {
			return nil, err
		}

		time.Sleep(1 * time.Second)

		home := "/root"
		if v, ok := os.LookupEnv("HOME"); ok {
			home = v
		}
		defer os.Remove(filepath.Join(home, ".vault-token"))
	}

	return createCA(ctx, vault)
}
