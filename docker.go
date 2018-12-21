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

// DockerAuth represents docker auth config
type DockerAuth struct {
	Username string
	Password string
}

// FetchDockerImageAsArchive downloads a docker image and saves it as an archive.
func FetchDockerImageAsArchive(ctx context.Context, image ContainerImage, archive string, auth *DockerAuth) error {
	policyContext, err := signature.NewPolicyContext(&signature.Policy{
		Default: []signature.PolicyRequirement{
			// NewPRInsecureAcceptAnything returns a new "insecureAcceptAnything" PolicyRequirement.
			signature.NewPRInsecureAcceptAnything(),
		},
	})
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

	namedTagged, ok := src.DockerReference().(reference.NamedTagged)
	if !ok {
		return errors.New("unexpected error; docker://<FullName> does not produce named-tagged reference")
	}

	var srcSystemCtx *types.SystemContext
	if auth != nil {
		srcSystemCtx = &types.SystemContext{
			DockerAuthConfig: &types.DockerAuthConfig{
				Username: auth.Username,
				Password: auth.Password,
			},
		}
	}

	options := &copy.Options{
		SourceCtx: srcSystemCtx,
		DestinationCtx: &types.SystemContext{
			DockerArchiveAdditionalTags: []reference.NamedTagged{namedTagged},
		},
	}

	err = RetryWithSleep(ctx, retryCount, time.Second,
		func(ctx context.Context) error {
			_, err = copy.Image(ctx, policyContext, dst, src, options)
			if err != nil {
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
