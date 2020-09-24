package neco

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/go-version"
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

// FindContainerImage finds a ContainerImage from name
func (a ArtifactSet) FindContainerImage(name string) (ContainerImage, error) {
	for _, img := range a.Images {
		if img.Name == name {
			return img, nil
		}
	}
	return ContainerImage{}, errors.New("no such container")
}

// FindDebianPackage finds a DebianPackage from name
func (a ArtifactSet) FindDebianPackage(name string) (DebianPackage, error) {
	for _, deb := range a.Debs {
		if deb.Name == name {
			return deb, nil
		}
	}
	return DebianPackage{}, errors.New("no such package")
}

// ContainerImage represents a Docker container image.
type ContainerImage struct {
	// Name is a unique name of this object.
	Name string

	// Repository is a docker repository name.
	Repository string

	// Tag is the image tag.
	Tag string

	// Private indicates that there is a private version of this image.
	Private bool
}

// ParseContainerImageName parses image name like "quay.io/cybozu/etcd:3.3.9-4"
func ParseContainerImageName(name string) (ContainerImage, error) {
	nametag := strings.Split(name, ":")
	if len(nametag) != 2 {
		return ContainerImage{}, errors.New("invalid image name: " + name)
	}

	return ContainerImage{
		Name:       path.Base(nametag[0]),
		Repository: nametag[0],
		Tag:        nametag[1],
	}, nil
}

// FullName returns full container image name.
// hasSecret should be true if the system has credentials to access private images.
func (c ContainerImage) FullName(hasSecret bool) string {
	if hasSecret && c.Private {
		return fmt.Sprintf("%s-secret:%s", c.Repository, c.Tag)
	}
	return fmt.Sprintf("%s:%s", c.Repository, c.Tag)
}

// MarshalGo formats the struct in Go syntax.
func (c ContainerImage) MarshalGo() string {
	return fmt.Sprintf("{Name: %q, Repository: %q, Tag: %q, Private: %t}",
		c.Name, c.Repository, c.Tag, c.Private)
}

// MajorVersion returns major version of this image.
func (c ContainerImage) MajorVersion() int {
	ver := version.Must(version.NewVersion(c.Tag))
	return ver.Segments()[0]
}

// DebianPackage represents a Debian package hosted in GitHub releases.
type DebianPackage struct {
	// Package name.
	Name string

	// Github Owner
	Owner string

	// GitHub repository.
	Repository string

	// GitHub releases (tag name).
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
	kernel := fmt.Sprintf("https://%s.release.flatcar-linux.net/amd64-usr/%s/flatcar_production_pxe.vmlinuz", c.Channel, c.Version)
	initrd := fmt.Sprintf("https://%s.release.flatcar-linux.net/amd64-usr/%s/flatcar_production_pxe_image.cpio.gz", c.Channel, c.Version)
	return kernel, initrd
}
