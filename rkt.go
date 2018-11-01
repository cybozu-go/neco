package neco

import (
	"context"
	"encoding/json"

	"github.com/cybozu-go/well"
)

// RktImage represents rkt image information
type RktImage struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FetchContainer fetches a container image
func FetchContainer(ctx context.Context, name string) error {
	img, err := CurrentArtifacts.FindContainerImage(name)
	if err != nil {
		return err
	}
	fullname := img.FullName()

	cmd := well.CommandContext(ctx, "rkt", "image", "list", "--format=json")
	data, err := cmd.Output()
	if err != nil {
		return err
	}
	var list []RktImage
	err = json.Unmarshal(data, &list)
	if err != nil {
		return err
	}
	for _, i := range list {
		if i.Name == fullname {
			return nil
		}
	}

	cmd = well.CommandContext(ctx, "rkt", "--insecure-options=image", "fetch", "--full", "docker://"+fullname)
	return cmd.Run()
}
