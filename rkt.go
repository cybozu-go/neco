package neco

import (
	"context"
	"encoding/json"
	"fmt"

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

// Bind represents a host bind mount rule.
type Bind struct {
	Name     string
	Source   string
	Dest     string
	ReadOnly bool
}

// Args returns command-line arguments for rkt.
func (b Bind) Args() []string {
	return []string{
		fmt.Sprintf("--volume=%s,kind=host,source=%s,readOnly=%v", b.Name, b.Source, b.ReadOnly),
		fmt.Sprintf("--mount=volume=%s,target=%s", b.Name, b.Dest),
	}
}

// RunContainer runs container in front.
func RunContainer(ctx context.Context, name string, binds []Bind, args []string) error {
	img, err := CurrentArtifacts.FindContainerImage(name)
	if err != nil {
		return err
	}

	rktArgs := []string{"run", "--pull-policy=never"}
	for _, b := range binds {
		rktArgs = append(rktArgs, b.Args()...)
	}
	rktArgs = append(rktArgs, img.FullName())
	rktArgs = append(rktArgs, args...)

	cmd := well.CommandContext(ctx, "rkt", rktArgs...)
	return cmd.Run()
}
