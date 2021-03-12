package neco

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	sabakan "github.com/cybozu-go/sabakan/v2/client"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"
)

const (
	testRoleDir = "ignitions/roles"
)

func testIgnitionTemplates(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	t := &sabakan.TemplateSource{}
	err = yaml.Unmarshal(data, t)
	if err != nil {
		return err
	}

	if len(t.Include) != 0 {
		abs, err := filepath.Abs(filepath.Join(filepath.Dir(path), t.Include))
		if err != nil {
			return err
		}

		_, err = os.Stat(abs)
		if err != nil {
			return err
		}

		err = testIgnitionTemplates(abs)
		if err != nil {
			return err
		}
	}

	if len(t.Passwd) != 0 {
		abs, err := filepath.Abs(filepath.Join(filepath.Dir(path), t.Passwd))
		if err != nil {
			return err
		}

		_, err = os.Stat(abs)
		if err != nil {
			return err
		}
	}

	if len(t.Files) != 0 {
		filesDir, err := filepath.Abs(filepath.Join(filepath.Dir(path), "files"))
		if err != nil {
			return err
		}

		var filelistInYAML []string
		for _, f := range t.Files {
			realPath := filepath.Join(filesDir, f)
			filelistInYAML = append(filelistInYAML, realPath)
		}
		sort.Strings(filelistInYAML)

		var filelistInDir []string
		err = filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			filelistInDir = append(filelistInDir, path)
			return nil
		})
		if err != nil {
			return err
		}

		sort.Strings(filelistInDir)

		if !reflect.DeepEqual(filelistInYAML, filelistInDir) {
			return fmt.Errorf("files in %s and file tree %s differ\n%v", path, filesDir, cmp.Diff(filelistInYAML, filelistInDir))
		}
	}

	// remote_files does not use in Neco.
	//if len(t.RemoteFiles) != 0 {
	//	...
	//}

	if len(t.Systemd) != 0 {
		systemdDir, err := filepath.Abs(filepath.Join(filepath.Dir(path), "systemd"))
		if err != nil {
			return err
		}

		var filelistInDir []string
		err = filepath.Walk(systemdDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			filelistInDir = append(filelistInDir, path)
			return nil
		})
		if err != nil {
			return err
		}

	OUTER:
		for _, f := range filelistInDir {
			for _, s := range t.Systemd {
				if f == filepath.Join(systemdDir, s.Name) {
					continue OUTER
				}
			}
			return fmt.Errorf("%s is not defined in %s", f, path)
		}
	}

	if len(t.Networkd) != 0 {
		networkdDir, err := filepath.Abs(filepath.Join(filepath.Dir(path), "networkd"))
		if err != nil {
			return err
		}

		var filelistInYAML []string
		for _, f := range t.Networkd {
			realPath := filepath.Join(networkdDir, f)
			filelistInYAML = append(filelistInYAML, realPath)
		}
		sort.Strings(filelistInYAML)

		var filelistInDir []string
		err = filepath.Walk(networkdDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			filelistInDir = append(filelistInDir, path)
			return nil
		})
		if err != nil {
			return err
		}

		sort.Strings(filelistInDir)

		if !reflect.DeepEqual(filelistInYAML, filelistInDir) {
			return fmt.Errorf("files in %s and file tree %s differ\n%v", path, networkdDir, cmp.Diff(filelistInYAML, filelistInDir))
		}
	}

	return nil
}

func TestNecoIgnitionTemplates(t *testing.T) {
	var siteYAMLs []string

	t.Parallel()
	err := filepath.Walk(testRoleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == "site.yml" {
			siteYAMLs = append(siteYAMLs, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, sy := range siteYAMLs {
		err := testIgnitionTemplates(sy)
		if err != nil {
			t.Error(err)
		}
	}
}
