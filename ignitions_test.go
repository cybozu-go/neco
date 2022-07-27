package neco

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	sabakan "github.com/cybozu-go/sabakan/v2/client"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"
)

const (
	testRoleDir = "ignitions/roles"
)

func checkForFlag(filestateInDir map[string]bool) error {
	for key, value := range filestateInDir {
		if !value {
			return fmt.Errorf("%s file is not included in the configuration file\n", key)
		}
	}
	return nil
}

func testIgnitionTemplates(path string, filestateInDir map[string]bool, final bool) error {
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
		err = testIgnitionTemplates(abs, filestateInDir, final)
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

		err = filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if _, ok := filestateInDir[path]; !ok {
				filestateInDir[path] = false
			}
			return nil
		})

		if err != nil {
			return err
		}

		for _, f := range filelistInYAML {
			if _, ok := filestateInDir[f]; ok {
				filestateInDir[f] = true
			} else {
				return fmt.Errorf("file in %s and file tree differ\n", f)
			}
		}

		if final {
			if err := checkForFlag(filestateInDir); err != nil {
				return err
			}
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

func isMaxTrial(roleCount int, count int) bool {
	return roleCount == count
}

func TestNecoIgnitionTemplates(t *testing.T) {
	var siteYAMLs []string
	roleCount := map[string]int{}

	t.Parallel()
	err := filepath.Walk(testRoleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if s, _ := regexp.MatchString(`site(-.*|)\.yml`, info.Name()); s {
			siteYAMLs = append(siteYAMLs, path)
			role := strings.Split(path, "/")[2]
			roleCount[role]++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var trialCount int
	filestateInDir := map[string]bool{}
	for _, sy := range siteYAMLs {
		trialCount++
		role := strings.Split(sy, "/")[2]

		final := isMaxTrial(roleCount[role], trialCount)
		err := testIgnitionTemplates(sy, filestateInDir, final)

		if err != nil {
			t.Fatal(err)
		}

		if final {
			trialCount = 0
			filestateInDir = map[string]bool{}
		}
	}
}
