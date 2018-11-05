package vault

import (
	"errors"

	"github.com/hashicorp/vault/api"
)

// Unseal unseals the vault server.
func Unseal(vc *api.Client, unsealKey string) error {
	st, err := vc.Sys().Unseal(unsealKey)
	if err != nil {
		return err
	}
	if st.Sealed {
		return errors.New("failed to unseal vault")
	}
	return nil
}
