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

// MarshalGo formats the struct in Go syntax.
func (c ContainerImage) MarshalGo() string {
	return fmt.Sprintf(`var %sImage = ContainerImage{Name: %q, Repository: %q, Tag: %q}`,
		c.Name, c.Name, c.Repository, c.Tag)
}

// DebianPackage represents a Debian package hosted in GitHub releases.
type DebianPackage struct {
	// Package name.
	Name string

	// GitHub repository.
	Repository string

	// GitHub releases.
	Release string
}

// MarshalGo formats the struct in Go syntax.
func (deb DebianPackage) MarshalGo() string {
	return fmt.Sprintf(`var %sImage = DebianPackage{Name: %q, Repository: %q, Release: %q}`,
		deb.Name, deb.Name, deb.Repository, deb.Release)
}

// CoreOSImage represents CoreOS Container Linux kernel and initrd images.
type CoreOSImage struct {
	Channel string
	Version string
}

// MarshalGo formats the struct in Go syntax.
func (c CoreOSImage) MarshalGo() string {
	return fmt.Sprintf(`var CurrentCoreOSImage = CoreOSImage{Channel: %q, Version: %q}`,
		c.Channel, c.Version)
}
