package neco

import (
	"fmt"
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
		t.Error(err)
	}

	sort.Strings(filelistInDir)

	fmt.Println(pretty.Compare(filelistInYAML, filelistInDir))
	if !reflect.DeepEqual(filelistInYAML, filelistInDir) {
		t.Errorf("\nyaml      %v\ndirectory %v", filelistInYAML, filelistInDir)
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
		t.Error(err)
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

func TestIgnitions(t *testing.T) {
	t.Run("common", testIgnitionsCommon)
	t.Run("systemd", testIgnitionsSystemd)
	//t.Run("site", testIgnitionsSite)
}
