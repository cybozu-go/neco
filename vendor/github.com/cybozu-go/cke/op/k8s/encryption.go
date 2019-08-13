package k8s

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/cybozu-go/cke"
)

const (
	encryptionConfigDir  = "/etc/kubernetes/apiserver"
	encryptionConfigFile = encryptionConfigDir + "/encryption.yml"
)

func getEncryptionSecret(ctx context.Context, inf cke.Infrastructure, key string) (string, error) {
	vc, err := inf.Vault()
	if err != nil {
		return "", err
	}

	secret, err := vc.Logical().Read(cke.K8sSecret)
	if err != nil {
		return "", err
	}
	if secret == nil {
		return "", errors.New("no encryption secrets for API server")
	}

	data, ok := secret.Data[key]
	if !ok {
		return "", errors.New("no secret data for " + key)
	}
	return data.(string), nil
}

func getEncryptionConfiguration(ctx context.Context, inf cke.Infrastructure) (*EncryptionConfiguration, error) {
	data, err := getEncryptionSecret(ctx, inf, "aescbc")
	if err != nil {
		return nil, err
	}

	aescfg := new(AESConfiguration)
	err = json.Unmarshal([]byte(data), aescfg)
	if err != nil {
		return nil, err
	}

	cfg := newEncryptionConfiguration()
	resources := []ResourceConfiguration{
		{
			Resources: []string{"secrets"},
			Providers: []ProviderConfiguration{
				{AESCBC: aescfg},
				{Identity: &struct{}{}},
			},
		},
	}
	cfg.Resources = resources

	return &cfg, nil
}
