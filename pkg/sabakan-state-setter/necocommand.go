package sss

import (
	"context"
	"os/exec"
)

// NecoCmdExecutor is interface for the neco command
type NecoCmdExecutor interface {
	TPMClear(ctx context.Context, serial string) ([]byte, error)
}

type necoCmdExecutor struct{}

func newNecoCmdExecutor() necoCmdExecutor {
	return necoCmdExecutor{}
}

func (necoCmdExecutor) TPMClear(ctx context.Context, serial string) ([]byte, error) {
	return exec.CommandContext(ctx, "neco", "tpm", "clear", "--force", serial).CombinedOutput()
}
