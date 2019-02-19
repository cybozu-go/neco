package sabakan

import (
	"regexp"
	"time"
)

const (
	// MaxImages is the maximum number of images that an index can hold.
	MaxImages = 5

	// ImageKernelFilename is a filename appear in TAR archive of an image.
	ImageKernelFilename = "kernel"

	// ImageInitrdFilename is a filename appear in TAR archive of an image.
	ImageInitrdFilename = "initrd.gz"
)

var (
	reValidImageID = regexp.MustCompile(`^[0-9a-zA-Z.-]+$`)
	reValidImageOS = regexp.MustCompile(`^[a-z0-9.]+$`)
)

// IsValidImageID returns true if id is valid as an image ID.
func IsValidImageID(id string) bool {
	return reValidImageID.MatchString(id)
}

// IsValidImageOS returns true if id is valid as OS.
func IsValidImageOS(os string) bool {
	return reValidImageOS.MatchString(os)
}

// Image represents a set of image files for iPXE boot.
type Image struct {
	ID     string    `json:"id"`
	Date   time.Time `json:"date"`
	URLs   []string  `json:"urls"`
	Exists bool      `json:"exists"`
}

// ImageIndex is a list of *Image.
type ImageIndex []*Image

// Append appends a new *Image to the index.
//
// If the index has MaxImages images, the oldest image will be discarded.
// ID of discarded images are returned in the second return value.
func (i ImageIndex) Append(img *Image) (ImageIndex, []string) {
	if len(i) < MaxImages {
		return append(i, img), nil
	}

	ndels := len(i) - MaxImages + 1
	dels := make([]string, ndels)
	for j := 0; j < ndels; j++ {
		dels[j] = i[j].ID
	}
	copy(i, i[ndels:])
	i[MaxImages-1] = img
	return i[0:MaxImages], dels
}

// Remove removes an image entry from the index.
func (i ImageIndex) Remove(id string) ImageIndex {
	newIndex := make(ImageIndex, 0, len(i))
	for _, img := range i {
		if img.ID == id {
			continue
		}
		newIndex = append(newIndex, img)
	}

	return newIndex
}

// Find an image whose ID is id.
//
// If no image can be found, this returns nil.
func (i ImageIndex) Find(id string) *Image {
	for _, img := range i {
		if img.ID == id {
			return img
		}
	}

	return nil
}
