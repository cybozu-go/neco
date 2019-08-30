package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	ign22 "github.com/coreos/ignition/config/v2_2/types"
	ign23 "github.com/coreos/ignition/config/v2_3/types"
	"github.com/vincent-petithory/dataurl"
	"sigs.k8s.io/yaml"
)

// SystemdUnit represents a systemd unit in Ignition template.
type SystemdUnit struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Mask    bool   `json:"mask"`
}

// RemoteFile represents a remote file.
type RemoteFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Mode *int   `json:"mode"`
}

// TemplateSource represents YAML/JSON source file of Ignition template.
type TemplateSource struct {
	Version     IgnitionVersion `json:"version"`
	Include     string          `json:"include"`
	Passwd      string          `json:"passwd"`
	Files       []string        `json:"files"`
	RemoteFiles []RemoteFile    `json:"remote_files"`
	Systemd     []SystemdUnit   `json:"systemd"`
	Networkd    []string        `json:"networkd"`
}

func loadSource(sourceFile, baseDir string) (*TemplateSource, string, error) {
	if !filepath.IsAbs(sourceFile) {
		sourceFile = filepath.Join(baseDir, sourceFile)
	}
	newBaseDir, err := filepath.EvalSymlinks(filepath.Dir(sourceFile))
	if err != nil {
		return nil, "", err
	}

	data, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		return nil, "", err
	}

	src := &TemplateSource{}
	err = yaml.Unmarshal(data, src)
	if err != nil {
		return nil, "", fmt.Errorf("invalid source YAML: %s: %v", sourceFile, err)
	}

	if src.Version == "" {
		src.Version = Ignition2_2 // for backward compatibility
	}
	return src, newBaseDir, nil
}

// BuildIgnitionTemplate constructs an IgnitionTemplate from source file.
func BuildIgnitionTemplate(sourceFile string, metadata map[string]interface{}) (*IgnitionTemplate, error) {
	cwd, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	src, baseDir, err := loadSource(sourceFile, cwd)
	if err != nil {
		return nil, err
	}

	tmpl := &IgnitionTemplate{
		Version:  src.Version,
		Metadata: metadata,
	}
	switch src.Version {
	case Ignition2_2:
		ign, err := buildTemplate2_2(src, baseDir)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(ign)
		if err != nil {
			return nil, err
		}
		tmpl.Template = json.RawMessage(data)
	case Ignition2_3:
		ign, err := buildTemplate2_3(src, baseDir)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(ign)
		if err != nil {
			return nil, err
		}
		tmpl.Template = json.RawMessage(data)
	default:
		return nil, errors.New("unsupported ignition spec: " + string(src.Version))
	}

	return tmpl, nil
}

func buildTemplate2_2(src *TemplateSource, baseDir string) (*ign22.Config, error) {
	var cfg *ign22.Config
	if src.Include == "" {
		cfg = &ign22.Config{}
	} else {
		parentSrc, parentBaseDir, err := loadSource(src.Include, baseDir)
		if err != nil {
			return nil, err
		}
		if parentSrc.Version != src.Version {
			return nil, errors.New("unmatched ignition version in " + src.Include)
		}
		cfg, err = buildTemplate2_2(parentSrc, parentBaseDir)
		if err != nil {
			return nil, err
		}
	}

	if src.Passwd != "" {
		passwdFile := src.Passwd
		if !filepath.IsAbs(passwdFile) {
			passwdFile = filepath.Join(baseDir, passwdFile)
		}

		data, err := ioutil.ReadFile(passwdFile)
		if err != nil {
			return nil, err
		}

		var passwd ign22.Passwd
		err = yaml.Unmarshal(data, &passwd)
		if err != nil {
			return nil, fmt.Errorf("invalid passwd YAML: %s: %v", passwdFile, err)
		}
		cfg.Passwd = passwd
	}

	for _, fname := range src.Files {
		// filepath.IsAbs is intentionally avoided to allow running clients on Windows.
		if !strings.HasPrefix(fname, "/") {
			return nil, errors.New("non-absolute filename: " + fname)
		}
		target := filepath.Join(baseDir, "files", fname)
		data, err := ioutil.ReadFile(target)
		if err != nil {
			return nil, err
		}
		fi, err := os.Stat(target)
		if err != nil {
			return nil, err
		}
		var file ign22.File
		file.Filesystem = "root"
		file.Path = fname
		file.Contents.Source = "data:," + dataurl.Escape(data)
		mode := int(fi.Mode().Perm())
		file.Mode = &mode
		cfg.Storage.Files = append(cfg.Storage.Files, file)
	}

	for _, remoteFile := range src.RemoteFiles {
		fname := remoteFile.Name
		// filepath.IsAbs is intentionally avoided to allow running clients on Windows.
		if !strings.HasPrefix(fname, "/") {
			return nil, errors.New("non-absolute filename: " + fname)
		}
		var file ign22.File
		file.Filesystem = "root"
		file.Path = fname
		file.Contents.Source = remoteFile.URL
		file.Mode = remoteFile.Mode
		cfg.Storage.Files = append(cfg.Storage.Files, file)
	}

	for _, netunit := range src.Networkd {
		target := filepath.Join(baseDir, "networkd", netunit)
		data, err := ioutil.ReadFile(target)
		if err != nil {
			return nil, err
		}

		var unit ign22.Networkdunit
		unit.Name = netunit
		unit.Contents = string(data)
		cfg.Networkd.Units = append(cfg.Networkd.Units, unit)
	}

	for _, sysunit := range src.Systemd {
		var unit ign22.Unit
		unit.Name = sysunit.Name
		if sysunit.Mask {
			unit.Mask = true
		} else {
			target := filepath.Join(baseDir, "systemd", sysunit.Name)
			data, err := ioutil.ReadFile(target)
			if err != nil {
				return nil, err
			}

			if sysunit.Enabled {
				unit.Enabled = &sysunit.Enabled
			}
			unit.Contents = string(data)
		}
		cfg.Systemd.Units = append(cfg.Systemd.Units, unit)
	}

	return cfg, nil
}

func buildTemplate2_3(src *TemplateSource, baseDir string) (*ign23.Config, error) {
	var cfg *ign23.Config
	if src.Include == "" {
		cfg = &ign23.Config{}
	} else {
		parentSrc, parentBaseDir, err := loadSource(src.Include, baseDir)
		if err != nil {
			return nil, err
		}
		if parentSrc.Version != src.Version {
			return nil, errors.New("unmatched ignition version in " + src.Include)
		}
		cfg, err = buildTemplate2_3(parentSrc, parentBaseDir)
		if err != nil {
			return nil, err
		}
	}

	if src.Passwd != "" {
		passwdFile := src.Passwd
		if !filepath.IsAbs(passwdFile) {
			passwdFile = filepath.Join(baseDir, passwdFile)
		}

		data, err := ioutil.ReadFile(passwdFile)
		if err != nil {
			return nil, err
		}

		var passwd ign23.Passwd
		err = yaml.Unmarshal(data, &passwd)
		if err != nil {
			return nil, fmt.Errorf("invalid passwd YAML: %s: %v", passwdFile, err)
		}
		cfg.Passwd = passwd
	}

	for _, fname := range src.Files {
		// filepath.IsAbs is intentionally avoided to allow running clients on Windows.
		if !strings.HasPrefix(fname, "/") {
			return nil, errors.New("non-absolute filename: " + fname)
		}
		target := filepath.Join(baseDir, "files", fname)
		data, err := ioutil.ReadFile(target)
		if err != nil {
			return nil, err
		}
		fi, err := os.Stat(target)
		if err != nil {
			return nil, err
		}
		var file ign23.File
		file.Filesystem = "root"
		file.Path = fname
		file.Contents.Source = "data:," + dataurl.Escape(data)
		mode := int(fi.Mode().Perm())
		file.Mode = &mode
		cfg.Storage.Files = append(cfg.Storage.Files, file)
	}

	for _, remoteFile := range src.RemoteFiles {
		fname := remoteFile.Name
		// filepath.IsAbs is intentionally avoided to allow running clients on Windows.
		if !strings.HasPrefix(fname, "/") {
			return nil, errors.New("non-absolute filename: " + fname)
		}
		var file ign23.File
		file.Filesystem = "root"
		file.Path = fname
		file.Contents.Source = remoteFile.URL
		file.Mode = remoteFile.Mode
		cfg.Storage.Files = append(cfg.Storage.Files, file)
	}

	for _, netunit := range src.Networkd {
		target := filepath.Join(baseDir, "networkd", netunit)
		data, err := ioutil.ReadFile(target)
		if err != nil {
			return nil, err
		}

		var unit ign23.Networkdunit
		unit.Name = netunit
		unit.Contents = string(data)
		cfg.Networkd.Units = append(cfg.Networkd.Units, unit)
	}

	for _, sysunit := range src.Systemd {
		var unit ign23.Unit
		unit.Name = sysunit.Name
		if sysunit.Mask {
			unit.Mask = true
		} else {
			target := filepath.Join(baseDir, "systemd", sysunit.Name)
			data, err := ioutil.ReadFile(target)
			if err != nil {
				return nil, err
			}

			if sysunit.Enabled {
				unit.Enabled = &sysunit.Enabled
			}
			unit.Contents = string(data)
		}
		cfg.Systemd.Units = append(cfg.Systemd.Units, unit)
	}

	return cfg, nil
}
