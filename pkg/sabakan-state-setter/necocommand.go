package sss

import (
	"bytes"
	"context"
	"os/exec"
)

// NecoCmdExecutor is interface for the neco command
type NecoCmdExecutor interface {
	TPMClear(ctx context.Context, serial string) error
}

type necoCmdExecutor struct{}

func newNecoCmdExecutor() necoCmdExecutor {
	return necoCmdExecutor{}
}

func execNeco(ctx context.Context, opts ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "neco", opts...)
	outBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	err := cmd.Run()
	return outBuf.String(), err
}

func (necoCmdExecutor) TPMClear(ctx context.Context, serial string) error {
	_, err := execNeco(ctx, "tpm", "clear", serial)
	return err
}
