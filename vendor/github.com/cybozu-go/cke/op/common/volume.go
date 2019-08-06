package common

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
)

type volumeCreateCommand struct {
	nodes   []*cke.Node
	volname string
}

// VolumeCreateCommand returns a Commander to create a volume on nodes.
func VolumeCreateCommand(nodes []*cke.Node, name string) cke.Commander {
	return volumeCreateCommand{nodes, name}
}

func (c volumeCreateCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	env := well.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := inf.Engine(n.Address)
		env.Go(func(ctx context.Context) error {
			return ce.VolumeCreate(c.volname)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c volumeCreateCommand) Command() cke.Command {
	return cke.Command{
		Name:   "volume-create",
		Target: c.volname,
	}
}

type volumeRemoveCommand struct {
	nodes   []*cke.Node
	volname string
}

// VolumeRemoveCommand returns a Commander to remove a volume on nodes.
func VolumeRemoveCommand(nodes []*cke.Node, name string) cke.Commander {
	return volumeRemoveCommand{nodes, name}
}

func (c volumeRemoveCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	env := well.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := inf.Engine(n.Address)
		env.Go(func(ctx context.Context) error {
			exists, err := ce.VolumeExists(c.volname)
			if err != nil {
				return err
			}
			if exists {
				return ce.VolumeRemove(c.volname)
			}
			return nil
		})
	}
	env.Stop()
	return env.Wait()
}

func (c volumeRemoveCommand) Command() cke.Command {
	return cke.Command{
		Name:   "volume-remove",
		Target: c.volname,
	}
}
