package neco

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/containers/image/copy"
	"github.com/containers/image/docker/reference"
	"github.com/containers/image/signature"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/cybozu-go/log"
)

// FetchDockerImageAsArchive downloads a docker image and saves it as an archive.
func FetchDockerImageAsArchive(ctx context.Context, image ContainerImage, archive string) error {
	// Is the default policy usable?
	policy, err := signature.DefaultPolicy(nil)
	if err != nil {
		return err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return err
	}
	defer policyContext.Destroy()

	src, err := alltransports.ParseImageName("docker://" + image.FullName())
	if err != nil {
		return err
	}

	dst, err := alltransports.ParseImageName("docker-archive:" + archive)
	if err != nil {
		return err
	}

	// Does this really succeed?
	namedTagged, ok := src.DockerReference().(reference.NamedTagged)
	if !ok {
		return errors.New("unexpected error; docker://<FullName> does not produce named-tagged reference")
	}

	options := &copy.Options{
		DestinationCtx: &types.SystemContext{
			DockerArchiveAdditionalTags: []reference.NamedTagged{namedTagged},
		},
	}

	err = RetryWithSleep(ctx, retryCount, time.Second,
		func(ctx context.Context) error {
			_, err = copy.Image(ctx, policyContext, dst, src, options)
			if err != nil {
				// no need?
				os.Remove(archive)
			}
			return err
		},
		func(err error) {
			log.Warn("docker: failed to copy a container image to an archive", map[string]interface{}{
				log.FnError: err,
				"image":     image.FullName(),
				"archive":   archive,
			})
		},
	)
	if err == nil {
		log.Info("docker: copied a container image", map[string]interface{}{
			"image":   image.FullName(),
			"archive": archive,
		})
	}
	return err
}
