package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	sabac "github.com/cybozu-go/sabakan/v2/client"
	ign22 "github.com/flatcar-linux/ignition/config/v2_2/types"
	"github.com/vincent-petithory/dataurl"
	"sigs.k8s.io/yaml"
)

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	return false, err
}

var (
	cluster string
)

func init() {
	flag.CommandLine.Usage = func() {
		o := flag.CommandLine.Output()
		fmt.Fprintf(o, "\nUsage: %s --cluster=<cluster> <role>\n", flag.CommandLine.Name())
	}
	flag.StringVar(&cluster, "cluster", "", "cluster flag")
}

func main() {
	err := buildConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

func buildConfig() error {
	flag.Parse()

	role := flag.Arg(0)
	if len(role) == 0 {
		return fmt.Errorf("please set role argument")
	}

	abs, err := filepath.Abs("../ignitions/roles")
	if err != nil {
		return err
	}

	path := filepath.Join(abs, role, fmt.Sprintf("site-%s.yml", cluster))
	exist, err := fileExists(path)
	if err != nil {
		return err
	}
	if !exist {
		path = filepath.Join(abs, role, "site.yml")
	}

	tmpl, err := sabac.BuildIgnitionTemplate(path, nil)
	if err != nil {
		return err
	}
	var cfg *ign22.Config
	if err := json.Unmarshal(tmpl.Template, &cfg); err != nil {
		return err
	}

	// unescape file contents source
	for i, file := range cfg.Storage.Files {
		source := file.FileEmbedded1.Contents.Source
		source = strings.ReplaceAll(source, "data:,", "")
		decodeContents, err := dataurl.UnescapeToString(source)
		if err != nil {
			return err
		}
		cfg.Storage.Files[i].FileEmbedded1.Contents.Source = decodeContents
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	err = saveConfig(role, filepath.Base(path), data)
	if err != nil {
		return err
	}
	return nil
}

func saveConfig(role string, filename string, data []byte) error {
	path := filepath.Join("/tmp/roles", role)
	if err := os.MkdirAll(path, 0766); err != nil {
		return err
	}
	path = filepath.Join(path, filename)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(data); err != nil {
		return err
	}
	fmt.Printf("Create a file in %s\n", path)

	return f.Sync()
}
