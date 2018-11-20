package sabakan

import (
	"context"
	"os"

	"github.com/cybozu-go/neco"
	"github.com/hashicorp/vault/api"
)

// InitLocal initialize sabakan on local machine
func InitLocal(ctx context.Context, vc *api.Client) error {
	err := os.MkdirAll(neco.SabakanDataDir, 0755)
	if err != nil {
		return err
	}
	return IssueCerts(ctx, vc)
}
