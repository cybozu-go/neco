package sss

import (
	"bytes"
	"context"
	"os/exec"
)

// NecoCmdExecutor is interface for the neco command
type NecoCmdExecutor interface {
	TPMClear(ctx context.Context, serial string) (string, error)
}

type necoCmdExecutor struct{}

func newNecoCmdExecutor() necoCmdExecutor {
	return necoCmdExecutor{}
}

func execNeco(ctx context.Context, opts ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "neco", opts...)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func (necoCmdExecutor) TPMClear(ctx context.Context, serial string) (string, error) {
	_, stderr, err := execNeco(ctx, "tpm", "clear", "--force", serial)
	return stderr, err
}
