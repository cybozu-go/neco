package cke

import (
	"bufio"
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/cybozu-go/neco"
	saba "github.com/cybozu-go/neco/progs/sabakan"
	"github.com/cybozu-go/neco/storage"
	sabakan "github.com/cybozu-go/sabakan/v2/client"
	"github.com/cybozu-go/well"
)

// UploadContents uploads contents to sabakan
func UploadContents(ctx context.Context, sabakanHTTP *http.Client, proxyHTTP *http.Client, version string, fetcher neco.ImageFetcher) error {
	client, err := sabakan.NewClient(neco.SabakanLocalEndpoint, sabakanHTTP)
	if err != nil {
		return err
	}

	images, err := GetCKEImages()
	if err != nil {
		return err
	}

	env := well.NewEnvironment(ctx)
	for _, img := range images {
		img := img
		env.Go(func(ctx context.Context) error {
			return saba.UploadImageAssets(ctx, img, client, fetcher)
		})
	}
	env.Stop()
	return env.Wait()
}

// GetCKEImages get images list from `ckecli images`
func GetCKEImages() ([]neco.ContainerImage, error) {
	output, err := exec.Command(neco.CKECLIBin, "images").Output()
	if err != nil {
		return nil, err
	}

	var images []neco.ContainerImage
	sc := bufio.NewScanner(bytes.NewReader(output))
	for sc.Scan() {
		img, err := neco.ParseContainerImageName(sc.Text())
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	err = sc.Err()
	if err != nil {
		return nil, err
	}
	return images, nil
}

// SetCKETemplate set cke template with overriding weights
func SetCKETemplate(ctx context.Context, st storage.Storage) error {
	cluster, err := neco.MyCluster()
	if err != nil {
		return err
	}

	ckeTemplate, err := ioutil.ReadFile(neco.CKETemplateFile)
	if err != nil {
		return err
	}

	newCkeTemplate, err := GenerateCKETemplate(ctx, st, cluster, ckeTemplate)
	if err != nil {
		return err
	}

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	_, err = f.Write(newCkeTemplate)
	if err != nil {
		return err
	}

	return well.CommandContext(ctx, neco.CKECLIBin, "sabakan", "set-template", f.Name()).Run()
}
