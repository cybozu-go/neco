package neco

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// ImageFetcher retrieves Docker image from registries.
type ImageFetcher struct {
	transport http.RoundTripper
	env       string
}

// NewImageFetcher creates a new ImageFetcher.
// `transport` must not be nil.  `env` must not empty string.
func NewImageFetcher(transport http.RoundTripper, env string) (ImageFetcher, error) {
	if env == "" {
		return ImageFetcher{}, fmt.Errorf("env has no value set")
	}

	return ImageFetcher{
		transport: transport,
		env:       env,
	}, nil
}

// GetTarball fetches an image from the registry and write it as a tarball.
// The tarball can be loaded into Docker with `docker load`.
func (f ImageFetcher) GetTarball(ctx context.Context, img ContainerImage, w io.Writer) error {
	ref, err := name.ParseReference(img.FullName(f.env == DevEnv || f.env == StagingEnv || f.env == ProdEnv))
	if err != nil {
		return err
	}

	rimg, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithContext(ctx), remote.WithJobs(1), remote.WithTransport(f.transport))
	if err != nil {
		return fmt.Errorf("failed to create remote image for %s: %w", img.Name, err)
	}

	return tarball.Write(ref, rimg, w)
}
