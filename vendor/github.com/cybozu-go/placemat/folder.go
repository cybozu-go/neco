package placemat

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
)

// DataFolderFileSpec represents a DataFolder's File definition in YAML
type DataFolderFileSpec struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
	File string `yaml:"file,omitempty"`
}

// DataFolderSpec represents a DataFolder definition in YAML
type DataFolderSpec struct {
	Kind  string               `yaml:"kind"`
	Name  string               `yaml:"name"`
	Dir   string               `yaml:"dir,omitempty"`
	Files []DataFolderFileSpec `yaml:"files,omitempty"`
}

// DataFolder represents a data folder configuration
type DataFolder struct {
	*DataFolderSpec
	dirPath string
}

// NewDataFolder creates DataFolder from DataFolderSpec.
func NewDataFolder(spec *DataFolderSpec) (*DataFolder, error) {
	folder := &DataFolder{
		DataFolderSpec: spec,
	}

	if spec.Name == "" {
		return nil, errors.New("data folder name is empty")
	}
	if spec.Dir == "" && len(spec.Files) == 0 {
		return nil, errors.New("invalid DataFolder spec: " + spec.Name)
	}
	if spec.Dir != "" && len(spec.Files) > 0 {
		return nil, errors.New("invalid DataFolder spec: " + spec.Name)
	}

	for _, file := range spec.Files {
		if file.Name == "" {
			return nil, errors.New("invalid DataFolder spec: " + spec.Name)
		}
		if file.URL == "" && file.File == "" {
			return nil, errors.New("invalid DataFolder spec: " + spec.Name)
		}
		if file.URL != "" && file.File != "" {
			return nil, errors.New("invalid DataFolder spec: " + spec.Name)
		}
	}

	return folder, nil
}

// Path returns the filesystem path to the directory having folder contents.
func (d *DataFolder) Path() string {
	return d.dirPath
}

// Prepare copies or downloads necessary files to prepare folder contents.
func (d *DataFolder) Prepare(ctx context.Context, baseDir string, c *cache) error {
	if len(d.Dir) != 0 {
		st, err := os.Stat(d.Dir)
		if err != nil {
			return err
		}
		if !st.IsDir() {
			return errors.New(d.Dir + " is not a directory")
		}
		absPath, err := filepath.Abs(d.Dir)
		if err != nil {
			return err
		}
		d.dirPath = absPath
		return nil
	}

	p := filepath.Join(baseDir, d.Name)
	err := os.MkdirAll(p, 0755)
	if err != nil {
		return err
	}

	for _, file := range d.Files {
		dstPath := filepath.Join(p, file.Name)
		if file.File != "" {
			err = writeToFile(file.File, dstPath, nil)
			if err != nil {
				return err
			}
			continue
		}

		u, err := url.Parse(file.URL)
		if err != nil {
			return err
		}
		err = downloadData(ctx, u, nil, c)
		if err != nil {
			return err
		}
		err = copyDownloadedData(u, dstPath, c)
		if err != nil {
			return err
		}
	}

	d.dirPath = p
	return nil
}
