package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
)

func render(src, dest string) error {
	s, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	data := string(s)

	var images []neco.ContainerImage
	images = append(images, neco.CurrentArtifacts.Images...)
	images = append(images, neco.SystemContainers...)
	for _, image := range images {
		data = strings.Replace(data, "%%"+image.Name+"%%", neco.ImageAssetName(image), -1)
	}

	sabakanImage, err := neco.CurrentArtifacts.FindContainerImage("sabakan")
	if err != nil {
		return err
	}
	data = strings.Replace(data, "%%cryptsetup%%", neco.CryptsetupAssetName(sabakanImage), -1)

	d := []byte(data)
	return ioutil.WriteFile(dest, d, 0644)
}

func main() {
	flag.Parse()
	well.LogConfig{}.Apply()

	if flag.NArg() != 2 {
		log.Error("Usage: fill-asset-name <source dir> <dest dir>", nil)
		os.Exit(1)
	}

	srcDir := flag.Arg(0)
	destDir := flag.Arg(1)

	_, err := os.Stat(destDir)
	if err == nil {
		log.ErrorExit(errors.New("dest dir exists"))
	}
	if !os.IsNotExist(err) {
		log.ErrorExit(err)
	}

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			log.Error("walk error", map[string]interface{}{
				log.FnError: walkErr,
				"path":      path,
			})
			return walkErr
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(destDir, rel)
		if info.IsDir() {
			return os.Mkdir(dest, 0755)
		}

		return render(path, dest)
	})
	if err != nil {
		log.ErrorExit(err)
	}
}
