package menu

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/andreyvit/diff"
)

func testE2E(t *testing.T) {
	dir, err := ioutil.TempDir("", "placemat-menu-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cmd := exec.Command("go", "run", "../pkg/placemat-menu/main.go", "-f", "example/menu.yml", "-o", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	assertTargets(t, "testdata", dir)
}

func assertFileEqual(t *testing.T, f1, f2 string) {
	content1, err := ioutil.ReadFile(f1)
	if err != nil {
		t.Fatal(err)
	}
	content2, err := ioutil.ReadFile(f2)
	if err != nil {
		t.Fatal(err)
	}
	if string(content1) != string(content2) {
		t.Errorf("unexpected file content: %s\n%v", f1, diff.LineDiff(string(content1), string(content2)))
	}
}

func assertTargets(t *testing.T, testdataDir, resultDir string) {
	targets := []string{
		"cluster.yml",
		"bird_core.conf",
		"bird_spine1.conf",
		"bird_spine2.conf",
		"bird_rack0-tor1.conf",
		"bird_rack0-tor2.conf",
		"bird_rack1-tor1.conf",
		"bird_rack1-tor2.conf",
		"seed_boot-0.yml",
		"seed_boot-1.yml",
		"sabakan/machines.json",
		"setup-iptables",
	}

	for _, f := range targets {
		f1 := filepath.Join(testdataDir, f)
		f2 := filepath.Join(resultDir, f)
		assertFileEqual(t, f1, f2)
	}
}

func TestE2E(t *testing.T) {
	testE2E(t)
}
