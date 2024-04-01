package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"text/template"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/hashicorp/go-version"
)

var imageRepos = []string{
	"ghcr.io/cybozu-go/coil",
	"ghcr.io/cybozu/bird",
	"ghcr.io/cybozu/chrony",
	"ghcr.io/cybozu/etcd",
	"ghcr.io/cybozu/promtail",
	"ghcr.io/cybozu-go/sabakan",
	"ghcr.io/cybozu/serf",
	"quay.io/cybozu/setup-hw",
	"ghcr.io/cybozu/squid",
	"ghcr.io/cybozu/squid-exporter",
	"ghcr.io/cybozu/vault",
	"quay.io/cybozu/cilium",
	"ghcr.io/cybozu/cilium-operator-generic",
	"ghcr.io/cybozu/hubble-relay",
	"ghcr.io/cybozu/cilium-certgen",
}

func imageName(repo name.Repository) string {
	t := strings.Split(repo.RepositoryStr(), "/")
	return t[len(t)-1]
}

var privateImages = map[string]bool{
	"quay.io/cybozu/setup-hw": true,
}

var debRepos = []string{
	"etcdpasswd",
}

var templ = template.Must(template.New("").Parse(artifactSetTemplate))

const osImageFeed = "https://www.flatcar.org/releases-json/releases-stable.json"

func render(w io.Writer, release bool, images []*neco.ContainerImage, debs []*neco.DebianPackage, osImage *neco.OSImage) error {
	var data struct {
		Tag     string
		Images  []*neco.ContainerImage
		Debs    []*neco.DebianPackage
		OSImage *neco.OSImage
	}

	if release {
		data.Tag = "release"
	} else {
		data.Tag = "!release"
	}

	data.Images = images
	data.Debs = debs
	data.OSImage = osImage

	return templ.Execute(w, data)
}

// Config defines the parameters for Generate.
type Config struct {
	// tag the generated source code as release or not
	Release bool
	Ignored *IgnoreConfig
}

// IgnoreConfig defines the ignored versions of components.
type IgnoreConfig struct {
	Images  []ignoreImageConfig   `json:"images"`
	Debs    []ignoreDebConfig     `json:"debs"`
	OSImage []ignoreOSImageConfig `json:"osImage"`
}

type ignoreImageConfig struct {
	Repository string   `json:"repository"`
	Versions   []string `json:"versions"`
}

type ignoreDebConfig struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
}

type ignoreOSImageConfig struct {
	Channel  string   `json:"channel"`
	Versions []string `json:"versions"`
}

func (c *IgnoreConfig) getImageVersions(repo string) []string {
	for _, img := range c.Images {
		if img.Repository == repo {
			return img.Versions
		}
	}
	return nil
}

func (c *IgnoreConfig) getDebVersions(name string) []string {
	for _, deb := range c.Debs {
		if deb.Name == name {
			return deb.Versions
		}
	}
	return nil
}

func (c *IgnoreConfig) getOSImageVersions() []string {
	for _, core := range c.OSImage {
		if core.Channel == "stable" {
			return core.Versions
		}
	}
	return nil
}

// Generate generates new artifacts.go contents and writes it to out.
func Generate(ctx context.Context, cfg Config, out io.Writer) error {
	images := make([]*neco.ContainerImage, len(imageRepos))
	for i, repoStr := range imageRepos {
		repo, err := name.NewRepository(repoStr)
		if err != nil {
			return err
		}

		img, err := getLatestImage(ctx, repo, cfg.Release, cfg.Ignored.getImageVersions(repoStr))
		if err != nil {
			return err
		}
		images[i] = img
	}

	debs := make([]*neco.DebianPackage, 0, len(debRepos))
	for _, name := range debRepos {
		deb, err := getLatestDeb(ctx, name, cfg.Ignored.getDebVersions(name))
		if err != nil {
			return err
		}
		if deb == nil {
			continue
		}
		debs = append(debs, deb)
	}

	osImage, err := getLatestOSImage(ctx, cfg.Ignored.getOSImageVersions())
	if err != nil {
		return err
	}

	return render(out, cfg.Release, images, debs, osImage)
}

func getLatestImage(ctx context.Context, repo name.Repository, release bool, ignoreVersions []string) (*neco.ContainerImage, error) {
	tags, err := remote.List(repo)
	if err != nil {
		return nil, err
	}

	versions := make([]*version.Version, 0, len(tags))
OUTER:
	for _, tag := range tags {
		if strings.Count(tag, ".") < 2 {
			// ignore branch tags such as "1.2"
			continue
		}
		for _, ignored := range ignoreVersions {
			if tag == ignored {
				continue OUTER
			}
		}
		v, err := version.NewVersion(tag)
		if err != nil {
			continue
		}
		if release && v.Prerelease() != "" {
			continue
		}
		versions = append(versions, v)
	}

	name := imageName(repo)
	if release {
		current, err := neco.CurrentArtifacts.FindContainerImage(name)
		if err != nil {
			return nil, err
		}
		major := current.MajorVersion()
		filteredVersions := make([]*version.Version, 0, len(versions))
		for _, ver := range versions {
			if ver.Segments()[0] == major {
				filteredVersions = append(filteredVersions, ver)
			}
		}
		versions = filteredVersions
	}

	sort.Sort(sort.Reverse(version.Collection(versions)))
	return &neco.ContainerImage{
		Name:       name,
		Repository: repo.Name(),
		Tag:        versions[0].Original(),
		Private:    privateImages[repo.Name()],
	}, nil
}

func getLatestDeb(ctx context.Context, name string, ignoreVersions []string) (*neco.DebianPackage, error) {
	client := neco.NewDefaultGitHubClient()
	releases, resp, err := client.Repositories.ListReleases(ctx, "cybozu-go", name, nil)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		log.Error("failed to get the latest GitHub release", map[string]interface{}{
			"owner":      "cybozu-go",
			"repository": name,
			log.FnError:  err,
		})
		return nil, err
	}
	var version string
OUTER:
	for _, release := range releases {
		if release.TagName == nil {
			continue
		}
		for _, ignored := range ignoreVersions {
			if *release.TagName == ignored {
				continue OUTER
			}
		}
		version = *release.TagName
		break
	}

	if version == "" {
		log.Error("no available version", map[string]interface{}{
			"owner":      "cybozu-go",
			"repository": name,
		})
		return nil, errors.New(name + ": no available version was found")
	}

	return &neco.DebianPackage{
		Name:       name,
		Owner:      "cybozu-go",
		Repository: name,
		Release:    version,
	}, nil
}

func getLatestOSImage(ctx context.Context, ignoreVersions []string) (*neco.OSImage, error) {
	resp, err := http.Get(osImageFeed)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get OSImage feed. status = %d", resp.StatusCode)
	}

	var feed map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&feed)
	if err != nil {
		return nil, err
	}

	versions := make([]*version.Version, 0, len(feed))
OUTER:
	for k := range feed {
		if k == "current" {
			continue
		}
		v, err := version.NewVersion(k)
		if err != nil {
			log.Error("bad version", map[string]interface{}{
				"version": k,
			})
			return nil, err
		}
		for _, ignored := range ignoreVersions {
			if k == ignored {
				continue OUTER
			}
		}
		versions = append(versions, v)
	}
	sort.Sort(sort.Reverse(version.Collection(versions)))
	return &neco.OSImage{
		Channel: "stable",
		Version: versions[0].Original(),
	}, nil
}
