package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/cybozu-go/neco"
	"github.com/hashicorp/vault/api"
	"github.com/howeyc/gopass"
)

// vaultClient returns an authorized Vault client.
//
// If "VAULT_TOKEN" environment variable is set, its value
// is used as the token to access Vault.  Otherwise, this will
// ask the user Vault username and password.
func vaultClient(lrn int) (*api.Client, error) {
	cfg := api.DefaultConfig()
	cfg.Address = fmt.Sprintf("https://%s:8200", neco.BootNode0IP(lrn).String())
	vc, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if vc.Token() != "" {
		return vc, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Vault username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	username = username[0 : len(username)-1]
	pass, err := gopass.GetPasswdPrompt("Vault password: ", false, os.Stdin, os.Stdout)
	if err != nil {
		return nil, err
	}
	password := string(pass)

	secret, err := vc.Logical().Write("/auth/userpass/login/"+username,
		map[string]interface{}{"password": password})
	if err != nil {
		return nil, err
	}

	vc.SetToken(secret.Auth.ClientToken)
	return vc, nil
}
