package neco

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/hashicorp/vault/api"
)

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
