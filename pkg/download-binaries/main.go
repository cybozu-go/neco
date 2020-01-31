package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	"github.com/google/go-github/v18/github"
	"github.com/hashicorp/go-version"
	"golang.org/x/oauth2"
)

const (
	teleportWindowsURL = "https://get.gravitational.com/teleport-v%s-windows-amd64-bin.zip"
	kubectlWindowsURL  = "https://storage.googleapis.com/kubernetes-release/release/v%s/bin/windows/amd64/kubectl.exe"
)

var outputDir = flag.String("dir", ".", "The download path. Create the dir if not present.")

func main() {
	flag.Parse()
	env := well.NewEnvironment(context.Background())

	_, err := os.Stat(*outputDir)
	if err != nil && !os.IsNotExist(err) {
		log.ErrorExit(err)
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(*outputDir, 0755)
		if err != nil {
			log.ErrorExit(err)
		}
	}
	env.Go(downloadTeleport)
	env.Go(downloadKubectl)
	env.Stop()
	err = env.Wait()
	if err != nil {
		log.ErrorExit(err)
	}
}

func downloadKubectl(ctx context.Context) error {
	ckeTag := getImageTag("cke")
	if len(ckeTag) == 0 {
		return errors.New("cke not found in artifacts")
	}
	splitTags := strings.Split(ckeTag, ".")
	if len(splitTags) != 3 {
		return errors.New("cke unexpected tag format:" + ckeTag)
	}
	k8sVersion := strings.Join(splitTags[:len(splitTags)-1], ".")

	var hc *http.Client
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		hc = oauth2.NewClient(ctx, ts)
	}
	gc := neco.NewGitHubClient(hc)
	releases, resp, err := gc.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fullVersion, err := getLatestPatchVersion(releases, k8sVersion)
	if err != nil {
		return err
	}
	log.Info("downloading kubectl...", map[string]interface{}{"version": fullVersion})
	return downloadFile(ctx, fmt.Sprintf(kubectlWindowsURL, fullVersion))
}

func getLatestPatchVersion(releases []*github.RepositoryRelease, vString string) (string, error) {
	latestPatchVersion, err := version.NewVersion(fmt.Sprintf("%s.0", vString))
	if err != nil {
		return "", err
	}
	for _, r := range releases {
		if r.GetTargetCommitish() != fmt.Sprintf("release-%s", vString) {
			continue
		}
		v, err := version.NewVersion(strings.TrimPrefix(r.GetTagName(), "v"))
		if err != nil {
			log.Error("failed convert tag to version", map[string]interface{}{
				"tag":       r.GetTagName(),
				log.FnError: err,
			})
			continue
		}
		if latestPatchVersion.LessThan(v) {
			latestPatchVersion = v
		}
	}
	return latestPatchVersion.String(), nil
}

func downloadTeleport(ctx context.Context) error {
	teleportTag := getImageTag("teleport")
	if len(teleportTag) == 0 {
		return errors.New("teleport not found in artifacts")
	}
	splitTags := strings.Split(teleportTag, ".")
	if len(splitTags) != 4 {
		return errors.New("teleport unexpected tag format:" + teleportTag)
	}
	teleportVersion := strings.Join(splitTags[:len(splitTags)-1], ".")
	log.Info("downloading teleport...", map[string]interface{}{"version": teleportVersion})
	return downloadFile(ctx, fmt.Sprintf(teleportWindowsURL, teleportVersion))
}

func downloadFile(ctx context.Context, url string) error {
	f, err := ioutil.TempFile(*outputDir, path.Base(url)+"-")
	if err != nil {
		return err
	}
	defer f.Close()

	client := &well.HTTPClient{Client: &http.Client{}}
	req, _ := http.NewRequest("GET", url, nil)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
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
