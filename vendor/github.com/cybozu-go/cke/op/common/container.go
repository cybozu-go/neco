package common

import (
	"context"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

type runContainerCommand struct {
	nodes     []*cke.Node
	name      string
	img       cke.Image
	opts      []string
	optsMap   map[string][]string
	params    cke.ServiceParams
	paramsMap map[string]cke.ServiceParams
	extra     cke.ServiceParams

	restart bool
}

// RunOption is a functional option for RunContainerCommand
type RunOption func(c *runContainerCommand)

// RunContainerCommand returns a Commander to run or restart a system container.
func RunContainerCommand(nodes []*cke.Node, name string, img cke.Image, opts ...RunOption) cke.Commander {
	c := &runContainerCommand{nodes: nodes, name: name, img: img}
	for _, f := range opts {
		f(c)
	}
	return c
}

// WithRestart returns RunOption to restart a container.
func WithRestart() RunOption {
	return func(c *runContainerCommand) { c.restart = true }
}

// WithOpts returns RunOption to set container engine options.
func WithOpts(opts []string) RunOption {
	return func(c *runContainerCommand) { c.opts = opts }
}

// WithOptsMap returns RunOption to set container engine options for each node.
func WithOptsMap(optsMap map[string][]string) RunOption {
	return func(c *runContainerCommand) { c.optsMap = optsMap }
}

// WithParams returns RunOption to set ServiceParams.
func WithParams(params cke.ServiceParams) RunOption {
	return func(c *runContainerCommand) { c.params = params }
}

// WithParamsMap returns RunOption to set ServiceParams for each node.
func WithParamsMap(paramsMap map[string]cke.ServiceParams) RunOption {
	return func(c *runContainerCommand) { c.paramsMap = paramsMap }
}

// WithExtra returns RunOption to set extra ServiceParams.
func WithExtra(params cke.ServiceParams) RunOption {
	return func(c *runContainerCommand) { c.extra = params }
}

func (c runContainerCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	env := well.NewEnvironment(ctx)
	for _, n := range c.nodes {
		n := n
		ce := inf.Engine(n.Address)
		env.Go(func(ctx context.Context) error {
			params, ok := c.paramsMap[n.Address]
			if !ok {
				params = c.params
			}
			opts, ok := c.optsMap[n.Address]
			if !ok {
				opts = c.opts
			}
			if c.restart {
				err := ce.Kill(c.name)
				if err != nil {
					return err
				}
			}
			exists, err := ce.Exists(c.name)
			if err != nil {
				return err
			}
			if exists {
				err = ce.Remove(c.name)
				if err != nil {
					return err
				}
			}
			return ce.RunSystem(c.name, c.img, opts, params, c.extra)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c runContainerCommand) Command() cke.Command {
	return cke.Command{
		Name:   "run-container",
		Target: c.name,
	}
}

type stopContainerCommand struct {
	node *cke.Node
	name string
}

// StopContainerCommand returns a Commander to stop a container on a node.
func StopContainerCommand(node *cke.Node, name string) cke.Commander {
	return stopContainerCommand{node, name}
}

func (c stopContainerCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	begin := time.Now()
	ce := inf.Engine(c.node.Address)
	exists, err := ce.Exists(c.name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	// Inspect returns ServiceStatus for the named container.
	statuses, err := ce.Inspect([]string{c.name})
	if err != nil {
		return err
	}
	if st, ok := statuses[c.name]; ok && st.Running {
		err = ce.Stop(c.name)
		if err != nil {
			return err
		}
	}
	err = ce.Remove(c.name)
	log.Info("stop container", map[string]interface{}{
		"container": c.name,
		"elapsed":   time.Now().Sub(begin).Seconds(),
	})
	return err
}

func (c stopContainerCommand) Command() cke.Command {
	return cke.Command{
		Name:   "stop-container",
		Target: c.name,
	}
}

type killContainersCommand struct {
	nodes []*cke.Node
	name  string
}

// KillContainersCommand returns a Commander to kill a container on nodes.
func KillContainersCommand(nodes []*cke.Node, name string) cke.Commander {
	return killContainersCommand{nodes, name}
}

func (c killContainersCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	begin := time.Now()
	env := well.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := inf.Engine(n.Address)
		env.Go(func(ctx context.Context) error {
			exists, err := ce.Exists(c.name)
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}
			statuses, err := ce.Inspect([]string{c.name})
			if err != nil {
				return err
			}
			if st, ok := statuses[c.name]; ok && st.Running {
				err = ce.Kill(c.name)
				if err != nil {
					return err
				}
			}
			return ce.Remove(c.name)
		})
	}
	env.Stop()
	err := env.Wait()
	log.Info("kill container", map[string]interface{}{
		"container": c.name,
		"elapsed":   time.Now().Sub(begin).Seconds(),
	})
	return err
}

func (c killContainersCommand) Command() cke.Command {
	return cke.Command{
		Name:   "kill-containers",
		Target: c.name,
	}
}
