package cke

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"os/exec"

	"github.com/cybozu-go/neco"
	saba "github.com/cybozu-go/neco/progs/sabakan"
	sabakan "github.com/cybozu-go/sabakan/v2/client"
	"github.com/cybozu-go/well"
)

// UploadContents uploads contents to sabakan
func UploadContents(ctx context.Context, sabakanHTTP *http.Client, proxyHTTP *http.Client, version string) error {
	client, err := sabakan.NewClient(neco.SabakanLocalEndpoint, sabakanHTTP)
	if err != nil {
		return err
	}

	output, err := exec.Command(neco.CKECLIBin, "images").Output()
	if err != nil {
		return err
	}

	var images []neco.ContainerImage
	sc := bufio.NewScanner(bytes.NewReader(output))
	for sc.Scan() {
		img, err := neco.ParseContainerImageName(sc.Text())
		if err != nil {
			return err
		}
		images = append(images, img)
	}
	err = sc.Err()
	if err != nil {
		return err
	}

	env := well.NewEnvironment(ctx)
	for _, img := range images {
		img := img
		env.Go(func(ctx context.Context) error {
			return saba.UploadImageAssets(ctx, img, client, nil)
		})
	}
	env.Stop()
	return env.Wait()
}
