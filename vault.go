package neco

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/vault/api"
	"golang.org/x/term"
)

func readPasswordFromStdTerminal(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return "", fmt.Errorf("stdin and stdout are not terminals")
	}

	fmt.Print(prompt)
	p, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(p), nil
}

// WaitVaultLeader waits for Vault to elect a new leader after restart.
//
// Vault wrongly recognizes that the old leader is still a leader after
// restarting all Vault servers at once.  This is probably because the
// leader information is stored in etcd and Vault references that data
// to determine the current leader.
//
// While a leader is not yet elected, still Vault servers forward requests
// to the old non-leader.  What's bad is that although the old leader denies
// the forwarded requests, Vault's Go client library cannot return error.
//
// Specifically, without this workaround, api.Client.Logical.Write() to
// issue certificates would return (nil, nil)!
func WaitVaultLeader(ctx context.Context, vc *api.Client) error {
	// disable request forwarding
	// see https://www.vaultproject.io/docs/concepts/ha.html#request-forwarding
	h := http.Header{}
	h.Set("X-Vault-No-Request-Forwarding", "true")
	vc.SetHeaders(h)

	for i := 0; i < 100; i++ {
		// We use Sys().ListAuth() because Sys().Leader() or Sys().Health()
		// does not help!
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

// VaultClient returns an authorized Vault client.
//
// If "VAULT_TOKEN" environment variable is set, its value
// is used as the token to access Vault.  Otherwise, this will
// ask the user Vault username and password.
func VaultClient(lrn int) (*api.Client, error) {
	cfg := api.DefaultConfig()
	cfg.Address = fmt.Sprintf("https://%s:8200", BootNode0IP(lrn).String())
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

	password, err := readPasswordFromStdTerminal("Vault password: ")
	if err != nil {
		return nil, err
	}

	secret, err := vc.Logical().Write("/auth/userpass/login/"+username,
		map[string]interface{}{"password": password})
	if err != nil {
		return nil, err
	}

	vc.SetToken(secret.Auth.ClientToken)
	return vc, nil
}
