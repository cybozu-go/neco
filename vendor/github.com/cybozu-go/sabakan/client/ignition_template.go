package client

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

const baseFileDir = "files"
const baseSystemdDir = "systemd"
const baseNetworkdDir = "networkd"

type systemd struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`
	Mask    bool   `yaml:"mask"`
}

type ignitionSource struct {
	Passwd   string    `yaml:"passwd"`
	Files    []string  `yaml:"files"`
	Systemd  []systemd `yaml:"systemd"`
	Networkd []string  `yaml:"networkd"`
	Include  string    `yaml:"include"`
}

type ignitionBuilder struct {
	baseDir     string
	ignition    map[string]interface{}
	loadedFiles map[string]bool
}

func newIgnitionBuilder(baseDir string) *ignitionBuilder {
	return &ignitionBuilder{
		baseDir:     baseDir,
		ignition:    make(map[string]interface{}),
		loadedFiles: make(map[string]bool),
	}
}

// AssembleIgnitionTemplate assemble ignition template from fname to w
func AssembleIgnitionTemplate(fname string, w io.Writer) error {
	absPath, err := filepath.Abs(fname)
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(absPath)
	builder := newIgnitionBuilder(baseDir)

	source, err := loadSource(absPath)
	if err != nil {
		return err
	}

	builder.ignition["ignition"] = map[string]interface{}{
		"version": "2.2.0",
	}
	err = builder.constructIgnitionYAML(source)
	if err != nil {
		return err
	}
	return yaml.NewEncoder(w).Encode(builder.ignition)
}

func loadSource(fname string) (*ignitionSource, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var source ignitionSource
	err = yaml.Unmarshal(data, &source)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

func (b *ignitionBuilder) constructIgnitionYAML(source *ignitionSource) error {
	if source.Include != "" {
		if b.loadedFiles[source.Include] {
			return fmt.Errorf("same file was included: %s", source.Include)
		}
		b.loadedFiles[source.Include] = true

		include, err := loadSource(filepath.Join(b.baseDir, source.Include))
		if err != nil {
			return err
		}

		childBaseDir := filepath.Dir(filepath.Join(b.baseDir, source.Include))
		childBuilder := ignitionBuilder{baseDir: childBaseDir, ignition: b.ignition, loadedFiles: b.loadedFiles}
		err = childBuilder.constructIgnitionYAML(include)
		if err != nil {
			return err
		}
	}
	if source.Passwd != "" {
		err := b.constructPasswd(source.Passwd)
		if err != nil {
			return err
		}
	}

	for _, file := range source.Files {
		err := b.constructFile(file)
		if err != nil {
			return err
		}
	}

	for _, s := range source.Systemd {
		err := b.constructSystemd(s)
		if err != nil {
			return err
		}
	}

	for _, n := range source.Networkd {
		err := b.constructNetworkd(n)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *ignitionBuilder) constructPasswd(passwd string) error {
	pf, err := os.Open(filepath.Join(b.baseDir, passwd))
	if err != nil {
		return err
	}
	defer pf.Close()
	passData, err := ioutil.ReadAll(pf)
	if err != nil {
		return err
	}

	var p interface{}
	err = yaml.Unmarshal(passData, &p)
	if err != nil {
		return err
	}
	b.ignition["passwd"] = p

	return nil
}

func (b *ignitionBuilder) constructFile(inputFile string) error {
	if !filepath.IsAbs(inputFile) {
		return errors.New("file source must be absolute path")
	}
	p := filepath.Join(b.baseDir, baseFileDir, inputFile)
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	mode := int(fi.Mode())

	storage, ok := b.ignition["storage"].(map[string]interface{})
	if !ok {
		storage = make(map[string]interface{})
	}
	files, _ := storage["files"].([]interface{})
	files = append(files, map[string]interface{}{
		"path":       inputFile,
		"filesystem": "root",
		"mode":       mode,
		"contents": map[string]interface{}{
			"source": string(data),
		},
	})

	storage["files"] = files
	b.ignition["storage"] = storage

	return nil
}

func (b *ignitionBuilder) constructSystemd(s systemd) error {
	if len(s.Name) == 0 {
		return errors.New("name: is not defined in systemd field")
	}

	systemd, ok := b.ignition["systemd"].(map[string]interface{})
	if !ok {
		systemd = make(map[string]interface{})
	}
	units, _ := systemd["units"].([]interface{})

	var unit map[string]interface{}

	if s.Mask {
		unit = map[string]interface{}{
			"name": s.Name,
			"mask": s.Mask,
		}
	} else {
		data, err := ioutil.ReadFile(filepath.Join(b.baseDir, baseSystemdDir, s.Name))
		if err != nil {
			return err
		}
		unit = map[string]interface{}{
			"name":     s.Name,
			"enabled":  s.Enabled,
			"contents": string(data),
		}
	}

	units = append(units, unit)
	systemd["units"] = units
	b.ignition["systemd"] = systemd

	return nil
}

func (b *ignitionBuilder) constructNetworkd(n string) error {
	f, err := os.Open(filepath.Join(b.baseDir, baseNetworkdDir, n))
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	networkd, ok := b.ignition["networkd"].(map[string]interface{})
	if !ok {
		networkd = make(map[string]interface{})
	}
	units, _ := networkd["units"].([]interface{})
	units = append(units, map[string]interface{}{
		"name":     n,
		"contents": string(data),
	})
	networkd["units"] = units
	b.ignition["networkd"] = networkd

	return nil
}
