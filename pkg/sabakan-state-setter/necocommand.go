package sss

import (
	"context"
	"os/exec"
)

// NecoCmdExecutor is interface for the neco command
type NecoCmdExecutor interface {
	PowerStop(ctx context.Context, serial string) ([]byte, error)
	PowerStatus(ctx context.Context, serial string) ([]byte, error)
	TPMClear(ctx context.Context, serial string) ([]byte, error)
}

type necoCmdExecutor struct{}

func newNecoCmdExecutor() necoCmdExecutor {
	return necoCmdExecutor{}
}

func (necoCmdExecutor) PowerStop(ctx context.Context, serial string) ([]byte, error) {
	return exec.CommandContext(ctx, "neco", "power", "stop", serial).CombinedOutput()
}

func (necoCmdExecutor) PowerStatus(ctx context.Context, serial string) ([]byte, error) {
	return exec.CommandContext(ctx, "neco", "power", "status", serial).CombinedOutput()
}

func (necoCmdExecutor) TPMClear(ctx context.Context, serial string) ([]byte, error) {
	return exec.CommandContext(ctx, "neco", "tpm", "clear", "--force", serial).CombinedOutput()
}
