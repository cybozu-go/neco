package setup

import (
	"context"
	"os/exec"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/hashicorp/vault/api"
)

const vaultPath = "/usr/local/bin/vault"

func prepareCA(ctx context.Context, isLeader bool, mylrn int, lrns []int) ([][]byte, error) {
	err := neco.RunContainer(ctx, "vault",
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/vault/install-tools"})
	if err != nil {
		return nil, err
	}

	cfg := api.DefaultConfig()
	cfg.Address = "http://127.0.0.1:8200"
	vault, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	vault.SetToken("cybozu")

	if isLeader {
		tmpCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		cmd := exec.CommandContext(tmpCtx, vaultPath, "server", "-dev",
			"-dev-listen-address=0.0.0.0:8200", "-dev-root-token-id=cybozu")
		err = cmd.Start()
		if err != nil {
			return nil, err
		}

		time.Sleep(1 * time.Second)
	}

	// TODO
	return nil, nil
}
