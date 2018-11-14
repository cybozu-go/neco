package setup

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/vault"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/hashicorp/vault/api"
)

const vaultPath = "/usr/local/bin/vault"

func writeFile(filename string, data string) error {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, []byte(data), 0644)
}

func dumpCertFiles(secret *api.Secret, caFile, certFile, keyFile string) error {
	err := writeFile(certFile, secret.Data["certificate"].(string))
	if err != nil {
		return err
	}
	err = writeFile(keyFile, secret.Data["private_key"].(string))
	if err != nil {
		return err
	}
	if caFile == "" {
		return nil
	}
	return writeFile(caFile, secret.Data["issuing_ca"].(string))
}

func setupLocalCerts(ctx context.Context, vault *api.Client, lrn int) error {
	for {
		secret, err := vault.Logical().Read("secret/bootstrap")
		if err == nil && secret != nil && len(secret.Data) > 0 {
			break
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(1 * time.Second):
		}
	}

	log.Info("prepare: setup local certs", nil)

	myname, err := os.Hostname()
	if err != nil {
		return err
	}
	myip := neco.BootNode0IP(lrn)

	bip, err := bastionIP()
	if err != nil {
		return err
	}

	secret, err := vault.Logical().Write(neco.CAServer+"/issue/system", map[string]interface{}{
		"common_name": myname,
		"alt_names":   "localhost",
		"ip_sans":     []string{"127.0.0.1", myip.String(), bip.String()},
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

	_, err = vault.Logical().Write(fmt.Sprintf("secret/bootstrap_done/%d", lrn),
		map[string]interface{}{"done": 1})
	if err != nil {
		return err
	}
	log.Info("prepare: end", nil)

	return nil
}

func createCA(ctx context.Context, vault *api.Client, mylrn int) ([]*api.Secret, error) {
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

	// mount v1 KV secret engine instead of v2 for easy operation
	// https://www.vaultproject.io/api/secret/kv/kv-v1.html
	err = vault.Sys().Unmount("secret")
	if err != nil {
		return nil, err
	}
	kv1 := &api.MountInput{Type: "kv", Options: map[string]string{"version": "1"}}
	err = vault.Sys().Mount("secret", kv1)
	if err != nil {
		return nil, err
	}

	_, err = vault.Logical().Write("secret/bootstrap", map[string]interface{}{
		"ready": "go",
	})
	if err != nil {
		return nil, err
	}

	return []*api.Secret{serverPem, peerPem, clientPem}, nil
}

func prepareCA(ctx context.Context, isLeader bool, mylrn int, lrns []int) ([]*api.Secret, error) {
	err := vault.InstallTools(ctx)
	if err != nil {
		return nil, err
	}

	cfg := api.DefaultConfig()
	cfg.Address = fmt.Sprintf("http://%s:8200", neco.BootNode0IP(lrns[0]).String())
	vc, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	vc.SetToken("cybozu")

	if !isLeader {
		return nil, setupLocalCerts(ctx, vc, mylrn)
	}

	tmpCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(tmpCtx, vaultPath, "server", "-dev",
		"-dev-listen-address=0.0.0.0:8200", "-dev-root-token-id=cybozu")
	err = cmd.Start()
	if err != nil {
		cancel()
		return nil, err
	}
	defer func() {
		cancel()
		cmd.Wait()

		home := os.Getenv("HOME")
		if home == "" {
			home = "/root"
		}
		os.Remove(filepath.Join(home, ".vault-token"))
	}()

	time.Sleep(1 * time.Second)

	log.Info("prepare: create CA", nil)
	pems, err := createCA(ctx, vc, mylrn)
	if err != nil {
		return nil, err
	}

	err = setupLocalCerts(ctx, vc, mylrn)
	if err != nil {
		return nil, err
	}

	log.Info("prepare: sync", nil)
	for _, lrn := range lrns {
		for {
			secret, err := vc.Logical().Read(fmt.Sprintf("secret/bootstrap_done/%d", lrn))
			if err == nil && secret != nil && len(secret.Data) > 0 {
				break
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1 * time.Second):
			}
		}
	}

	return pems, nil
}

func setupVault(ctx context.Context, mylrn int, lrns []int) error {
	f, err := os.OpenFile(neco.VaultConfFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	err = vault.GenerateConf(f, mylrn, lrns)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}

	g, err := os.OpenFile(neco.ServiceFile(neco.VaultService), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer g.Close()

	err = vault.GenerateService(g)
	if err != nil {
		return err
	}
	err = g.Sync()
	if err != nil {
		return err
	}

	err = neco.StartService(ctx, neco.VaultService)
	if err != nil {
		return err
	}

	log.Info("vault: installed", nil)
	return nil
}

func bootVault(ctx context.Context, pems []*api.Secret, ec *clientv3.Client) (*api.Client, error) {
	cfg := api.DefaultConfig()
	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	req := &api.InitRequest{
		SecretShares:    1,
		SecretThreshold: 1,
	}
	resp, err := client.Sys().Init(req)
	if err != nil {
		return nil, err
	}

	unsealKey := resp.KeysB64[0]
	rootToken := resp.RootToken
	client.SetToken(rootToken)

	err = vault.Unseal(client, unsealKey)
	if err != nil {
		return nil, err
	}

	// output audit logs to stdout that should go to journald
	auditOpts := &api.EnableAuditOptions{
		Type: "file",
		Options: map[string]string{
			"file_path": "stdout",
		},
	}
	err = client.Sys().EnableAuditWithOptions("stdout", auditOpts)
	if err != nil {
		return nil, err
	}

	for i, ca := range []string{neco.CAServer, neco.CAEtcdPeer, neco.CAEtcdClient} {
		err := client.Sys().Mount(ca, &api.MountInput{
			Type: "pki",
			Config: api.MountConfigInput{
				MaxLeaseTTL:     neco.TTL100Year,
				DefaultLeaseTTL: neco.TTL10Year,
			},
		})
		if err != nil {
			return nil, err
		}
		cert := pems[i].Data["certificate"].(string)
		_, err = client.Logical().Write(ca+"/config/ca", map[string]interface{}{
			"pem_bundle": cert,
		})
		if err != nil {
			return nil, err
		}
	}

	_, err = client.Logical().Write(neco.CAServer+"/roles/system", map[string]interface{}{
		"ttl":            neco.TTL10Year,
		"max_ttl":        neco.TTL10Year,
		"client_flag":    false,
		"allow_any_name": true,
	})
	if err != nil {
		return nil, err
	}
	_, err = client.Logical().Write(neco.CAEtcdPeer+"/roles/system", map[string]interface{}{
		"ttl":            neco.TTL10Year,
		"max_ttl":        neco.TTL10Year,
		"allow_any_name": true,
	})
	if err != nil {
		return nil, err
	}
	_, err = client.Logical().Write(neco.CAEtcdClient+"/roles/system", map[string]interface{}{
		"ttl":            neco.TTL10Year,
		"max_ttl":        neco.TTL10Year,
		"server_flag":    false,
		"allow_any_name": true,
	})
	if err != nil {
		return nil, err
	}
	_, err = client.Logical().Write(neco.CAEtcdClient+"/roles/human", map[string]interface{}{
		"ttl":            "2h",
		"max_ttl":        "24h",
		"server_flag":    false,
		"allow_any_name": true,
	})
	if err != nil {
		return nil, err
	}

	// add policies for admin
	err = client.Sys().PutPolicy("admin", vault.AdminPolicy())
	if err != nil {
		return nil, err
	}
	err = client.Sys().PutPolicy("ca-admin", vault.CAAdminPolicy())
	if err != nil {
		return nil, err
	}

	opt := &api.EnableAuthOptions{
		Type: "approle",
	}
	err = client.Sys().EnableAuthWithOptions("approle", opt)
	if err != nil {
		return nil, err
	}

	// store unseal key and root token in etcd
	st := storage.NewStorage(ec)
	err = st.PutVaultRootToken(ctx, rootToken)
	if err != nil {
		return nil, err
	}
	err = st.PutVaultUnsealKey(ctx, unsealKey)
	if err != nil {
		return nil, err
	}

	log.Info("vault: booted", nil)
	return client, nil
}

func waitVault(ctx context.Context, ec *clientv3.Client) (string, error) {
	st := storage.NewStorage(ec)

	for {
		unsealKey, err := st.GetVaultUnsealKey(ctx)
		switch err {
		case nil:
			return unsealKey, nil
		case storage.ErrNotFound:
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(1 * time.Second):
			}
			continue
		default:
			return "", err
		}
	}
}

func reissueCerts(ctx context.Context, vc *api.Client, mylrn int) error {
	myname, err := os.Hostname()
	if err != nil {
		return err
	}
	myip := neco.BootNode0IP(mylrn)
	log.Info("reissue: server cert", nil)

	secret, err := vc.Logical().Write(neco.CAServer+"/issue/system", map[string]interface{}{
		"common_name": myname,
		"alt_names":   "localhost",
		"ip_sans":     []string{"127.0.0.1", myip.String()},
	})
	if err != nil {
		return err
	}
	err = dumpCertFiles(secret, "", neco.ServerCertFile, neco.ServerKeyFile)
	if err != nil {
		return err
	}

	log.Info("reissue: etcd peer cert", nil)

	// peer certificate must have valid IP SANs for authentication.
	// https://coreos.com/etcd/docs/3.3.1/op-guide/security.html#notes-for-tls-authentication
	bip, err := bastionIP()
	if err != nil {
		return err
	}
	secret, err = vc.Logical().Write(neco.CAEtcdPeer+"/issue/system", map[string]interface{}{
		"common_name":          myname,
		"ip_sans":              []string{myip.String(), bip.String()},
		"exclude_cn_from_sans": true,
	})
	if err != nil {
		return err
	}
	err = dumpCertFiles(secret, "", neco.EtcdPeerCertFile, neco.EtcdPeerKeyFile)
	if err != nil {
		return err
	}

	log.Info("reissue: etcd client cert for Vault", nil)
	secret, err = vc.Logical().Write(neco.CAEtcdClient+"/issue/system", map[string]interface{}{
		"common_name":          "vault",
		"exclude_cn_from_sans": true,
	})
	if err != nil {
		return err
	}
	return dumpCertFiles(secret, "", neco.VaultCertFile, neco.VaultKeyFile)
}

// waitVaultLeader waits for Vault to elect a new leader after restart.
//
// Vault wrongly recognizes that the old leader is still a leader after
// rebstarting all Vault servers at once.  This is probablly because the
// leader information is stored in etcd and Vault references that data
// to determine the current leader.
//
// While a leader is not yet elected, still Vault servers forward requests
// to the old non-leader.  What's bad is that although the old leader denies
// the forwarded requests, Vault's Go client library cannot return error.
//
// Specifically, without this workaround, api.Client.Logical.Write() to
// issue certificates would return (nil, nil)!
func waitVaultLeader(ctx context.Context, vc *api.Client) error {
	// disable request forwarding
	// see https://www.vaultproject.io/docs/concepts/ha.html#request-forwarding
	h := http.Header{}
	h.Set("X-Vault-No-Request-Forwarding", "true")
	vc.SetHeaders(h)

	for i := 0; i < 100; i++ {
		// We use Sys().ListAuth() because Sys().Leader() or Sys().Health()
		/// does not help!
		resp, err := vc.Sys().ListAuth()
		if err == nil && resp != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	return errors.New("vault leader is not elected")
}

func revokeRootToken(ctx context.Context, vc *api.Client, ec *clientv3.Client) error {
	err := vc.Auth().Token().RevokeSelf("")
	if err != nil {
		return err
	}
	return storage.NewStorage(ec).DeleteVaultRootToken(ctx)
}
