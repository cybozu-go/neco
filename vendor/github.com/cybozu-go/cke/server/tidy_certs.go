package server

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	vault "github.com/hashicorp/vault/api"
)

// TidyExpiredCertificates call tidy endpoints of Vault API
func (c Controller) TidyExpiredCertificates(ctx context.Context, client *vault.Client, ca string) error {
	tidyParams := make(map[string]interface{})
	tidyParams["tidy_cert_store"] = true
	tidyParams["tidy_revocation_list"] = true
	tidyParams["safety_buffer"] = (time.Minute).String()
	res, err := client.Logical().Write(path.Join(ca, "tidy"), tidyParams)
	if err != nil {
		return err
	}

	log.Debug(path.Join(ca, "tidy")+" is called", map[string]interface{}{
		"tidy_params": tidyParams,
		"ca":          ca,
		"res":         res,
	})

	// TODO: More appropriate error detection.
	// Vault client does not provide an interface to detect whether errors have occurred or not.
	if len(res.Warnings) == 0 {
		log.Warn("may be failed to tidy certs, since an empty message is returned", map[string]interface{}{
			"tidy_params": tidyParams,
			"ca":          ca,
			"res":         res,
		})
		return errors.New("may be failed to tidy certs, since an empty message is returned")
	}
	// TODO: More appropriate error detection.
	// Currently, use this message. https://github.com/hashicorp/vault/blob/975db34faf38f5bf564d13da38e141975d9f0fe3/builtin/credential/approle/path_tidy_user_id.go#L240
	if res.Warnings[0] != "Tidy operation successfully started. Any information from the operation will be printed to Vault's server logs." {
		log.Warn("may be failed to tidy certs, since the expected warning message is not found", map[string]interface{}{
			"tidy_params":         tidyParams,
			"ca":                  ca,
			"vault_resp_warnings": strings.Join(res.Warnings, ", "),
		})
		return errors.New("failed to tidy certs: " + strings.Join(res.Warnings, ", "))
	}

	log.Info("invoked vault tidy", map[string]interface{}{
		"tidy_params":         tidyParams,
		"ca":                  ca,
		"vault_resp_warnings": strings.Join(res.Warnings, ", "),
	})

	return nil
}
