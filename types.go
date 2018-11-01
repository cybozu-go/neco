package neco

import (
	"fmt"
)

// ArtifactSet represents a set of artifacts.
type ArtifactSet struct {
	// Container image list
	Images []ContainerImage

	// Debian package list
	Debs []DebianPackage

	// CoreOS image version
	CoreOS CoreOSImage
}

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
	return fmt.Sprintf("{Name: %q, Repository: %q, Tag: %q}",
		c.Name, c.Repository, c.Tag)
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
	return fmt.Sprintf("{Name: %q, Owner: %q, Repository: %q, Release: %q}",
		deb.Name, deb.Owner, deb.Repository, deb.Release)
}

// CoreOSImage represents CoreOS Container Linux kernel and initrd images.
type CoreOSImage struct {
	Channel string
	Version string
}

// MarshalGo formats the struct in Go syntax.
func (c CoreOSImage) MarshalGo() string {
	return fmt.Sprintf("CoreOSImage{Channel: %q, Version: %q}",
		c.Channel, c.Version)
}

// URLs returns kernel and initrd URLs.
func (c CoreOSImage) URLs() (string, string) {
	kernel := fmt.Sprintf("https://%s.release.core-os.net/amd64-usr/%s/coreos_production_pxe.vmlinuz", c.Channel, c.Version)
	initrd := fmt.Sprintf("https://%s.release.core-os.net/amd64-usr/%s/coreos_production_pxe_image.cpio.gz", c.Channel, c.Version)
	return kernel, initrd
}
