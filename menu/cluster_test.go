package menu

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/google/go-cmp/cmp"
)

func TestGenerate(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "placemat-menu-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outputDir)

	f, err := os.Open("example/menu.yml")
	if err != nil {
		t.Errorf("failed to file open: %v", err)
	}
	defer f.Close()

	inputDir, err := filepath.Abs("example")
	if err != nil {
		t.Errorf("failed to inputDir: %v", err)
	}

	menu, err := Parse(bufio.NewReader(f), inputDir)
	if err != nil {
		t.Errorf("failed to parse: %v", err)
	}

	cluster, err := NewCluster(menu)
	if err != nil {
		t.Errorf("failed to NewCluster: %v", err)
	}

	opt := &GenerateOption{
		ChronyTag: "999.999.999",
	}
	if err := cluster.Generate(inputDir, outputDir, opt); err != nil {
		t.Errorf("failed to Generate: %v", err)
	}

	assertTargets(t, "testdata", outputDir)
	assertClusterSpec(t, "testdata", outputDir)
}

func assertClusterSpec(t *testing.T, testdataDir, outputDir string) {
	r := bytes.NewReader(injectSabakanDirToExpectedClusterYaml(t, testdataDir, outputDir))
	expected, err := types.Parse(r)
	if err != nil {
		t.Fatal(err)
	}

	f2, err := os.Open(filepath.Join(outputDir, "cluster.yml"))
	if err != nil {
		t.Fatal(err)
	}

	actual, err := types.Parse(f2)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(expected, actual) {
		t.Errorf("unexpected cluster: %v", cmp.Diff(expected, actual))
	}
}

func injectSabakanDirToExpectedClusterYaml(t *testing.T, testdataDir, outputDir string) []byte {
	input := filepath.Join(testdataDir, "cluster.yml")
	f1, err := os.Open(input)
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()

	content, err := io.ReadAll(f1)
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := template.New(input).Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}

	sabakanDir, err := filepath.Abs(filepath.Join(outputDir, "sabakan"))
	if err != nil {
		t.Fatal(err)
	}
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, sabakanDir); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func assertTargets(t *testing.T, testdataDir, resultDir string) {
	targets := []string{
		"sabakan/machines.json",
		"bird_core.conf",
		"bird_rack0-tor1.conf",
		"bird_rack0-tor2.conf",
		"bird_rack1-tor1.conf",
		"bird_rack1-tor2.conf",
		"bird_spine1.conf",
		"bird_spine2.conf",
		"chrony-ign.yml",
		"machines.yml",
		"network.yml",
		"seed_boot-0.yml",
		"seed_boot-1.yml",
		"setup-default-gateway-external",
		"setup-default-gateway-operation",
		"setup-iptables",
		"setup-iptables-spine",
		"squid.conf",
	}

	for _, f := range targets {
		f1 := filepath.Join(testdataDir, f)
		f2 := filepath.Join(resultDir, f)
		assertFileEqual(t, f1, f2)
	}
}

func assertFileEqual(t *testing.T, f1, f2 string) {
	content1, err := os.ReadFile(f1)
	if err != nil {
		t.Fatal(err)
	}
	content2, err := os.ReadFile(f2)
	if err != nil {
		t.Fatal(err)
	}
	if string(content1) != string(content2) {
		t.Errorf("unexpected file content: %s\n%v", f1, cmp.Diff(string(content1), string(content2)))
	}
}
