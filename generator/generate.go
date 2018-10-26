package generator

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/containers/image/docker"
	"github.com/containers/image/types"
	"github.com/cybozu-go/neco"
	"github.com/hashicorp/go-version"
)

const (
	sabakanRepo = "//quay.io/cybozu/sabakan"
)

// Generate generates new artifasts.go contents and writes it to out.
func Generate(ctx context.Context, release bool, out io.Writer) error {
	_, err := sabakanImage(ctx)
	return err
}

func sabakanImage(ctx context.Context) (*neco.ContainerImage, error) {
	ref, err := docker.ParseReference(sabakanRepo)
	if err != nil {
		return nil, err
	}

	tags, err := docker.GetRepositoryTags(ctx, &types.SystemContext{}, ref)
	if err != nil {
		return nil, err
	}

	versions := make([]*version.Version, len(tags))
	for i, tag := range tags {
		v, err := version.NewVersion(tag)
		if err != nil {
			return nil, err
		}
		versions[i] = v
	}
	sort.Sort(sort.Reverse(version.Collection(versions)))
	fmt.Println(versions[0].Original())
	return nil, nil
}
