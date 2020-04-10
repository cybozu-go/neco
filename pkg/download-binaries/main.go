package main

import (
	"archive/zip"
	"bufio"
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
	v1 "k8s.io/api/apps/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const (
	zipFileName = "operation-cli-windows-amd64.zip"

	teleportWindowsURL = "https://get.gravitational.com/teleport-v%s-windows-amd64-bin.zip"
	teleportLinuxURL   = "https://get.gravitational.com/teleport-v%s-linux-amd64-bin.tar.gz"

	kubectlWindowsURL = "https://storage.googleapis.com/kubernetes-release/release/v%s/bin/windows/amd64/kubectl.exe"
	kubectlLinuxURL   = "https://storage.googleapis.com/kubernetes-release/release/v%s/bin/linux/amd64/kubectl"

	argoCDWindowsURL = "https://github.com/argoproj/argo-cd/releases/download/v%s/argocd-windows-amd64.exe"
	argoCDLinuxURL   = "https://github.com/argoproj/argo-cd/releases/download/v%s/argocd-linux-amd64"
)

var outputDir = flag.String("dir", ".", "The output directory. Create the directory if not present.")

type downloader struct {
	gh *github.Client
}

func main() {
	flag.Parse()
	ctx := context.Background()

	d := newDownloader(ctx)
	kubectlVersion, err := d.fetchKubectlVersion(ctx)
	if err != nil {
		log.ErrorExit(err)
	}
	teleportVersion, err := d.getTeleportVersion()
	if err != nil {
		log.ErrorExit(err)
	}
	argoCDTag, err := d.fetchArgoCDTag(ctx)
	if err != nil {
		log.ErrorExit(err)
	}

	err = prepareOutputDir()
	if err != nil {
		log.ErrorExit(err)
	}

	urls := []string{
		fmt.Sprintf(teleportWindowsURL, teleportVersion),
		fmt.Sprintf(teleportLinuxURL, teleportVersion),
		fmt.Sprintf(kubectlWindowsURL, kubectlVersion),
		fmt.Sprintf(kubectlLinuxURL, kubectlVersion),
		fmt.Sprintf(argoCDWindowsURL, argoCDTag),
		fmt.Sprintf(argoCDLinuxURL, argoCDTag),
	}

	env := well.NewEnvironment(ctx)
	for _, u := range urls {
		env.Go(generateDownloadFile(u))
	}

	env.Stop()
	err = env.Wait()
	if err != nil {
		log.ErrorExit(err)
	}

	files := []string{}
	for _, u := range urls {
		files = append(files, filepath.Base(u))
	}
	defer func() {
		for _, filename := range files {
			err = os.Remove(filepath.Join(*outputDir, filename))
			if err != nil {
				log.ErrorExit(err)
			}
		}
	}()

	err = createZip(files)
	if err != nil {
		log.ErrorExit(err)
	}
}

func createZip(files []string) error {
	log.Info("compressing", map[string]interface{}{"output": zipFileName})
	newZipFile, err := ioutil.TempFile(*outputDir, "zip-")
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	w := zip.NewWriter(newZipFile)
	defer w.Close()

	for _, filename := range files {
		err := func(filename string) error {
			src, err := os.Open(filepath.Join(*outputDir, filename))
			if err != nil {
				return err
			}
			defer src.Close()

			info, err := src.Stat()
			if err != nil {
				return err
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			header.Name = filename
			header.Method = zip.Deflate

			f, err := w.CreateHeader(header)
			if err != nil {
				return err
			}

			_, err = io.Copy(f, src)
			return err
		}(filename)
		if err != nil {
			return err
		}
	}

	return os.Rename(newZipFile.Name(), filepath.Join(*outputDir, zipFileName))
}

func prepareOutputDir() error {
	_, err := os.Stat(*outputDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(*outputDir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateDownloadFile(url string) func(context.Context) error {
	return func(ctx context.Context) error {
		log.Info("downloading ...", map[string]interface{}{"url": url})
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
}

func newDownloader(ctx context.Context) downloader {
	var hc *http.Client
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		hc = oauth2.NewClient(ctx, ts)
	}
	return downloader{neco.NewGitHubClient(hc)}
}

func (d *downloader) fetchKubectlVersion(ctx context.Context) (string, error) {
	log.Info("fetching kubectl version from github...", map[string]interface{}{})
	ckeTag := getImageTag("cke")
	if len(ckeTag) == 0 {
		return "", errors.New("cke not found in artifacts")
	}
	splitTags := strings.Split(ckeTag, ".")
	if len(splitTags) != 3 {
		return "", errors.New("cke unexpected tag format:" + ckeTag)
	}
	k8sVersion := strings.Join(splitTags[:len(splitTags)-1], ".")

	releases, resp, err := d.gh.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	fullVersion, err := getLatestPatchVersionFromReleases(releases, k8sVersion)
	if err != nil {
		return "", err
	}
	return fullVersion, nil
}

func (d *downloader) getTeleportVersion() (string, error) {
	teleportTag := getImageTag("teleport")
	if len(teleportTag) == 0 {
		return "", errors.New("teleport not found in artifacts")
	}
	splitTags := strings.Split(teleportTag, ".")
	if len(splitTags) != 4 {
		return "", errors.New("teleport unexpected tag format:" + teleportTag)
	}
	teleportVersion := strings.Join(splitTags[:len(splitTags)-1], ".")
	return teleportVersion, nil
}

func (d *downloader) fetchArgoCDTag(ctx context.Context) (string, error) {
	log.Info("fetching ArgoCD tag from github...", map[string]interface{}{})
	const argoCDUpstreamFilePath = "argocd/base/upstream/install.yaml"
	buf, err := d.gh.Repositories.DownloadContents(ctx, "cybozu-go", "neco-apps", argoCDUpstreamFilePath, &github.RepositoryContentGetOptions{Ref: "release"})
	if err != nil {
		return "", err
	}
	defer buf.Close()
	y := k8sYaml.NewYAMLReader(bufio.NewReader(buf))
	for {
		data, err := y.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		var d v1.Deployment
		err = yaml.Unmarshal(data, &d)
		if err != nil {
			continue
		}
		if d.Name != "argocd-server" {
			continue
		}
		for _, c := range d.Spec.Template.Spec.Containers {
			if c.Name == "argocd-server" {
				split := strings.Split(c.Image, ":")
				if len(split) != 2 {
					return "", errors.New("unexpected image format: " + c.Image)
				}
				return split[1], nil
			}
		}
	}
	return "", errors.New("argocd-server deployment not found")
}

func getImageTag(name string) (url string) {
	for _, img := range neco.CurrentArtifacts.Images {
		if img.Name == name {
			return img.Tag
		}
	}
	return ""
}

func getLatestPatchVersionFromReleases(releases []*github.RepositoryRelease, vString string) (string, error) {
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
