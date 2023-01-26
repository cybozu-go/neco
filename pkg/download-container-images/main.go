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

	"github.com/cybozu-go/neco"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: download-container-images DOWNLOAD_DIR")
	os.Exit(2)
}

func main() {
	if len(os.Args) != 2 {
		usage()
	}
	downloadDir := os.Args[1]

	for _, img := range neco.CurrentArtifacts.Images {
		fullName := img.FullName(true)

		fmt.Printf("\n*** downloading %s\n", fullName)
		cmd := exec.Command("docker", "pull", fullName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("docker pull failed: %v", err)
			return
		}

		// setup-hw-secret will be saved to cybozu-setup-hw-x.x.x.img (i.e. without "-secret"). it is intended.
		imgTarName := filepath.Join(downloadDir, fmt.Sprintf("cybozu-%s-%s.img", img.Name, img.Tag))
		fmt.Printf("*** saving %s to %s\n", fullName, imgTarName)
		cmd = exec.Command("docker", "save", "-o", imgTarName, fullName)
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
