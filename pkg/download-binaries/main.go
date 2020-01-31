package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/neco"
)

const teleportWindowsURL = "https://get.gravitational.com/teleport-v%s-windows-amd64-bin.zip"

var outputDir = flag.String("dir", ".", "The download path. Create the dir if not present.")

func main() {
	flag.Parse()

	_, err := os.Stat(*outputDir)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(*outputDir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	teleportTag := getImageTag("teleport")
	if len(teleportTag) == 0 {
		log.Fatal("teleport not found in artifacts")
	}
	splitTags := strings.Split(teleportTag, ".")
	if len(splitTags) != 4 {
		log.Fatal("teleport unexpected tag format:" + teleportTag)
	}
	teleportVersion := strings.Join(splitTags[:len(splitTags)-1], ".")
	err = downloadFile(fmt.Sprintf(teleportWindowsURL, teleportVersion))
	if err != nil {
		log.Fatal(err)
	}
}

func downloadFile(url string) error {
	f, err := ioutil.TempFile(*outputDir, "")
	if err != nil {
		return err
	}
	defer f.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(*outputDir, path.Base(url)))
}

func getImageTag(name string) (url string) {
	for _, img := range neco.CurrentArtifacts.Images {
		if img.Name == name {
			return img.Tag
		}
	}
	return ""
}
