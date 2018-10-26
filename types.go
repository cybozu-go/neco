package neco

import "fmt"

// ContainerImage represents a Docker container image.
type ContainerImage struct {
	// A unique name for this container image.
	Name string

	// Repository is a docker repository name.
	Repository string

	// Image tag.
	Tag string
}

// URL returns docker image URL.
func (c ContainerImage) URL() string {
	return fmt.Sprintf("docker://%s:%s:", c.Repository, c.Tag)
}

// DebianPackage represents a Debian package hosted in GitHub releases.
type DebianPackage struct {
	Name       string
	Repository string
	Release    string
}

// CoreOSImage represents CoreOS Container Linux kernel and initrd images.
type CoreOSImage struct {
	Channel string
	Version string
}
