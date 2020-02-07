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
	zipFileName        = "operation-cli-windows-amd64.zip"
	teleportWindowsURL = "https://get.gravitational.com/teleport-v%s-windows-amd64-bin.zip"
	kubectlWindowsURL  = "https://storage.googleapis.com/kubernetes-release/release/v%s/bin/windows/amd64/kubectl.exe"
	argoCDWindowsURL   = "https://github.com/argoproj/argo-cd/releases/download/v%s/argocd-windows-amd64"
)

var outputDir = flag.String("dir", ".", "The output directory. Create the directory if not present.")

type downloader struct {
	gh *github.Client
}

func main() {
	flag.Parse()
	ctx := context.Background()
	env := well.NewEnvironment(ctx)

	err := prepareOutputDir()
	if err != nil {
		log.ErrorExit(err)
	}

	d := newDownloader(ctx)
	env.Go(d.downloadKubectl)
	env.Go(d.downloadTeleport)
	// TODO: Opt in downloadArgoCD after releasing windows binaries at https://github.com/argoproj/argo-cd/releases/
	// env.Go(d.downloadArgoCD)

	env.Stop()
	err = env.Wait()
	if err != nil {
		log.ErrorExit(err)
	}

	zipFiles, err := filepath.Glob("*.zip")
	if err != nil {
		log.ErrorExit(err)
	}
	exeFiles, err := filepath.Glob("*.exe")
	if err != nil {
		log.ErrorExit(err)
	}
	files := append(zipFiles, exeFiles...)
	// TODO: Opt in after releasing windows binaries at https://github.com/argoproj/argo-cd/releases/
	//files = append(files, filepath.Base(argoCDWindowsURL))

	err = createZip(files)
	if err != nil {
		log.ErrorExit(err)
	}
	for _, filename := range files {
		err = os.Remove(filename)
		if err != nil {
			log.ErrorExit(err)
		}
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
			src, err := os.Open(filename)
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

func (d *downloader) downloadKubectl(ctx context.Context) error {
	ckeTag := getImageTag("cke")
	if len(ckeTag) == 0 {
		return errors.New("cke not found in artifacts")
	}
	splitTags := strings.Split(ckeTag, ".")
	if len(splitTags) != 3 {
		return errors.New("cke unexpected tag format:" + ckeTag)
	}
	k8sVersion := strings.Join(splitTags[:len(splitTags)-1], ".")

	releases, resp, err := d.gh.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fullVersion, err := getLatestPatchVersionFromReleases(releases, k8sVersion)
	if err != nil {
		return err
	}
	log.Info("downloading kubectl...", map[string]interface{}{"version": fullVersion})
	return downloadFile(ctx, fmt.Sprintf(kubectlWindowsURL, fullVersion))
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

func (d *downloader) downloadTeleport(ctx context.Context) error {
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

func (d *downloader) downloadArgoCD(ctx context.Context) error {
	argoCDTag, err := d.getArgoCDTag(ctx)
	if err != nil {
		return err
	}
	log.Info("downloading argocd...", map[string]interface{}{"version": argoCDTag})
	return downloadFile(ctx, fmt.Sprintf(argoCDWindowsURL, argoCDTag))
}

func (d *downloader) getArgoCDTag(ctx context.Context) (string, error) {
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
