package neco

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	sabakan "github.com/cybozu-go/sabakan/v2/client"
	"github.com/kylelemons/godebug/pretty"
	"gopkg.in/yaml.v2"
)

const (
	commonYAML     = "ignitions/common/common.yml"
	testFilesDir   = "ignitions/common/files"
	testSystemdDir = "ignitions/common/systemd"
	testRoleDir    = "ignitions/roles"
)

func testIgnitionsCommon(t *testing.T) {
	t.Parallel()

	data, err := ioutil.ReadFile(commonYAML)
	if err != nil {
		t.Fatal(err)
	}

	src := &sabakan.TemplateSource{}
	err = yaml.Unmarshal(data, src)
	if err != nil {
		t.Fatal(err)
	}

	var filelistInYAML []string
	for _, f := range src.Files {
		realPath := filepath.Join(testFilesDir, f)
		filelistInYAML = append(filelistInYAML, realPath)
	}
	sort.Strings(filelistInYAML)

	var filelistInDir []string
	err = filepath.Walk(testFilesDir, func(path string, info os.FileInfo, err error) error {
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
		t.Fatal(err)
	}

	sort.Strings(filelistInDir)

	if !reflect.DeepEqual(filelistInYAML, filelistInDir) {
		t.Errorf("files in common.yml and file tree is not same\n%v", pretty.Compare(filelistInYAML, filelistInDir))
	}
}

func testIgnitionsSystemd(t *testing.T) {
	t.Parallel()

	data, err := ioutil.ReadFile(commonYAML)
	if err != nil {
		t.Fatal(err)
	}

	src := &sabakan.TemplateSource{}
	err = yaml.Unmarshal(data, src)
	if err != nil {
		t.Fatal(err)
	}

	var systemdInYAML []string
	for _, s := range src.Systemd {
		realPath := filepath.Join(testSystemdDir, s.Name)
		systemdInYAML = append(systemdInYAML, realPath)
	}

	var filelistInDir []string
	err = filepath.Walk(testSystemdDir, func(path string, info os.FileInfo, err error) error {
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
		t.Fatal(err)
	}

	for _, f := range filelistInDir {
		found := false
		for _, s := range src.Systemd {
			if f == filepath.Join(testSystemdDir, s.Name) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s is not defined in %s", f, commonYAML)
		}
	}
}

func testIgnitionsRole(t *testing.T) {
	t.Parallel()

	var siteYAMLs []string
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
		data, err := ioutil.ReadFile(sy)
		if err != nil {
			t.Fatal(err)
		}

		src := &sabakan.TemplateSource{}
		err = yaml.Unmarshal(data, src)
		if err != nil {
			t.Fatal(err)
		}

		abs, err := filepath.Abs(filepath.Join(filepath.Dir(sy), src.Include))
		if err != nil {
			t.Fatal(err)
		}

		_, err = os.Stat(abs)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestIgnitions(t *testing.T) {
	t.Run("common", testIgnitionsCommon)
	t.Run("systemd", testIgnitionsSystemd)
	t.Run("role", testIgnitionsRole)
}
