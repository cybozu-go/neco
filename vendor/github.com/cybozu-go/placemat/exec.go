package placemat

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

func execCommands(ctx context.Context, commands [][]string) error {
	for _, cmds := range commands {
		c := well.CommandContext(ctx, cmds[0], cmds[1:]...)
		c.Severity = log.LvDebug
		err := c.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func execCommandsForce(commands [][]string) error {
	ctx := context.Background()

	var firstError error
	for _, cmds := range commands {
		c := well.CommandContext(ctx, cmds[0], cmds[1:]...)
		c.Severity = log.LvDebug
		err := c.Run()
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}
