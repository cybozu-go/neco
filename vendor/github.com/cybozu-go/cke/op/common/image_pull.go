package common

import (
	"context"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

type imagePullCommand struct {
	nodes []*cke.Node
	img   cke.Image
}

const (
	pullMaxRetry     = 3
	pullWaitDuration = 10 * time.Second
)

// ImagePullCommand returns a Commander to pull an image on nodes.
func ImagePullCommand(nodes []*cke.Node, img cke.Image) cke.Commander {
	return imagePullCommand{nodes, img}
}

func (c imagePullCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	env := well.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := inf.Engine(n.Address)
		env.Go(func(ctx context.Context) error {
			var err error
			for i := 0; i < pullMaxRetry; i++ {
				err = ce.PullImage(c.img)
				if err == nil {
					return nil
				}

				log.Warn("failed to pull image", map[string]interface{}{
					"image":     c.img.Name(),
					log.FnError: err,
				})
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(pullWaitDuration):
				}
			}
			return err
		})
	}
	env.Stop()
	return env.Wait()
}

func (c imagePullCommand) Command() cke.Command {
	return cke.Command{
		Name:   "image-pull",
		Target: c.img.Name(),
	}
}
