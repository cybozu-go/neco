package neco

import (
	"fmt"
	"strings"
)

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
	return fmt.Sprintf("var %sImage = ContainerImage{Name: %q, Repository: %q, Tag: %q}\n",
		strings.Title(c.Name), c.Name, c.Repository, c.Tag)
}

// DebianPackage represents a Debian package hosted in GitHub releases.
type DebianPackage struct {
	// Package name.
	Name string

	// Github Owner
	Owner string

	// GitHub repository.
	Repository string

	// GitHub releases.
	Release string
}

// MarshalGo formats the struct in Go syntax.
func (deb DebianPackage) MarshalGo() string {
	return fmt.Sprintf("var %sPackage = DebianPackage{Name: %q, Owner: %q, Repository: %q, Release: %q}\n",
		strings.Title(deb.Name), deb.Name, deb.Owner, deb.Repository, deb.Release)
}

// CoreOSImage represents CoreOS Container Linux kernel and initrd images.
type CoreOSImage struct {
	Channel string
	Version string
}

// MarshalGo formats the struct in Go syntax.
func (c CoreOSImage) MarshalGo() string {
	return fmt.Sprintf("var CoreOS = CoreOSImage{Channel: %q, Version: %q}\n",
		c.Channel, c.Version)
}
