package k8s

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/cybozu-go/cke"
	apiserverv1 "k8s.io/apiserver/pkg/apis/config/v1"
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

func getEncryptionConfiguration(ctx context.Context, inf cke.Infrastructure) (*apiserverv1.EncryptionConfiguration, error) {
	data, err := getEncryptionSecret(ctx, inf, "aescbc")
	if err != nil {
		return nil, err
	}

	aescfg := new(apiserverv1.AESConfiguration)
	err = json.Unmarshal([]byte(data), aescfg)
	if err != nil {
		return nil, err
	}

	return &apiserverv1.EncryptionConfiguration{
		Resources: []apiserverv1.ResourceConfiguration{
			{
				Resources: []string{"secrets"},
				Providers: []apiserverv1.ProviderConfiguration{
					{AESCBC: aescfg},
					{Identity: &apiserverv1.IdentityConfiguration{}},
				},
			},
		},
	}, nil
}
