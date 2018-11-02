package setup

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
)

func testCreateCA(t *testing.T) {
	ctx := context.Background()
	cfg := api.DefaultConfig()
	cfg.Address = "http://127.0.0.1:8200"
	vault, err := api.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	vault.SetToken("cybozu")

	tmpCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(tmpCtx, vaultPath, "server", "-dev",
		"-dev-listen-address=0.0.0.0:8200", "-dev-root-token-id=cybozu")
	err = cmd.Start()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	defer func() {
		cancel()
		cmd.Wait()
	}()

	time.Sleep(1 * time.Second)

	secrets, err := createCA(ctx, vault, 1)
	if err != nil {
		t.Fatal(err)
	}

	for _, secret := range secrets {
		data, err := json.Marshal(secret)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(data))
	}
}

func TestVault(t *testing.T) {
	if _, err := os.Stat(vaultPath); err != nil {
		t.Skip(err)
	}
	t.Run("createCA", testCreateCA)
}
