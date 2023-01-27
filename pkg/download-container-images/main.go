package main

/*
 * download-container-images
 *
 * Pull container images in artifacts.go and save them to tar file.
 * This is used for setting up a new cluster in case that the boot servers do not have direct nor proxied internet connectivity.
 */

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/neco"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: download-container-images DOWNLOAD_DIR")
	os.Exit(2)
}

type ImageSpec struct {
	FullName string
	TarName  string
}

func main() {
	if len(os.Args) != 2 {
		usage()
	}
	downloadDir := os.Args[1]

	specs := []ImageSpec{}
	for _, img := range neco.CurrentArtifacts.Images {
		specs = append(specs, ImageSpec{
			FullName: img.FullName(true),
			// setup-hw-secret will be saved to cybozu-setup-hw-x.x.x.img (i.e. without "-secret"). it is intended.
			TarName: fmt.Sprintf("cybozu-%s-%s.img", img.Name, img.Tag),
		})
	}
	for _, fullName := range cke.AllImages() {
		names := strings.Split(fullName, ":")
		pathComponents := strings.Split(names[0], "/")
		specs = append(specs, ImageSpec{
			FullName: fullName,
			TarName:  fmt.Sprintf("cybozu-%s-%s.img", pathComponents[2], names[1]),
		})
	}

	for _, img := range specs {
		fmt.Printf("\n*** downloading %s\n", img.FullName)
		cmd := exec.Command("docker", "pull", img.FullName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("docker pull failed: %v", err)
			return
		}

		imgTarName := filepath.Join(downloadDir, img.TarName)
		fmt.Printf("*** saving %s to %s\n", img.FullName, imgTarName)
		cmd = exec.Command("docker", "save", "-o", imgTarName, img.FullName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			fmt.Printf("docker save failed: %v", err)
			return
		}
		fmt.Printf("ok\n")
	}
}
