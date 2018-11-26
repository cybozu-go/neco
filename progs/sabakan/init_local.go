package sabakan

import (
	"context"
	"os"

	"github.com/cybozu-go/neco"
	"github.com/hashicorp/vault/api"
)

// InitLocal initialize sabakan on local machine
func InitLocal(ctx context.Context, vc *api.Client) error {
	return os.MkdirAll(neco.SabakanDataDir, 0755)
}
