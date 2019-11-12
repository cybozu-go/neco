package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/containers/image/docker"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/hashicorp/go-version"
	"golang.org/x/oauth2"
)

var quayRepos = []string{
	"cke",
	"etcd",
	"setup-hw",
	"sabakan",
	"serf",
	"vault",
	"coil",
	"squid",
	"teleport",
}

var privateImages = map[string]bool{
	"setup-hw": true,
}

var debRepos = []string{
	"etcdpasswd",
}

var templ = template.Must(template.New("").Parse(artifactSetTemplate))

const coreOSFeed = "https://coreos.com/releases/releases-stable.json"

func render(w io.Writer, release bool, images []*neco.ContainerImage, debs []*neco.DebianPackage, coreos *neco.CoreOSImage) error {
	var data struct {
		Tag    string
		Images []*neco.ContainerImage
		Debs   []*neco.DebianPackage
		CoreOS *neco.CoreOSImage
	}

	if release {
		data.Tag = "release"
	} else {
		data.Tag = "!release"
	}

	data.Images = images
	data.Debs = debs
	data.CoreOS = coreos

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
	Images []ignoreImageConfig  `json:"images"`
	Debs   []ignoreDebConfig    `json:"debs"`
	CoreOS []ignoreCoreOSConfig `json:"coreOS"`
}

type ignoreImageConfig struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
}

type ignoreDebConfig struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
}

type ignoreCoreOSConfig struct {
	Channel  string   `json:"channel"`
	Versions []string `json:"versions"`
}

func (c *IgnoreConfig) getImageVersions(name string) []string {
	for _, img := range c.Images {
		if img.Name == name {
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

func (c *IgnoreConfig) getCoreOSVersions() []string {
	for _, core := range c.CoreOS {
		if core.Channel == "stable" {
			return core.Versions
		}
	}
	return nil
}

// Generate generates new artifacts.go contents and writes it to out.
func Generate(ctx context.Context, cfg Config, out io.Writer) error {
	images := make([]*neco.ContainerImage, len(quayRepos))
	for i, name := range quayRepos {
		img, err := getLatestImage(ctx, name, cfg.Release, cfg.Ignored.getImageVersions(name))
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

	coreos, err := getLatestCoreOS(ctx, cfg.Ignored.getCoreOSVersions())
	if err != nil {
		return err
	}

	return render(out, cfg.Release, images, debs, coreos)
}

func getLatestImage(ctx context.Context, name string, release bool, ignoreVersions []string) (*neco.ContainerImage, error) {
	ref, err := docker.ParseReference("//quay.io/cybozu/" + name)
	if err != nil {
		return nil, err
	}

	tags, err := docker.GetRepositoryTags(ctx, nil, ref)
	if err != nil {
		log.Error("failed to get the latest docker image tag", map[string]interface{}{
			"repository": "quay.io/cybozu/" + name,
			log.FnError:  err,
		})
		return nil, err
	}
	versions := make([]*version.Version, 0, len(tags))
	for _, tag := range tags {
		if strings.Count(tag, ".") < 2 {
			// ignore branch tags such as "1.2"
			continue
		}
		for _, ignored := range ignoreVersions {
			if tag == ignored {
				continue
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
		Repository: "quay.io/cybozu/" + name,
		Tag:        versions[0].Original(),
		Private:    privateImages[name],
	}, nil
}

func getLatestDeb(ctx context.Context, name string, ignoreVersions []string) (*neco.DebianPackage, error) {
	var hc *http.Client
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		hc = oauth2.NewClient(ctx, ts)
	}
	client := neco.NewGitHubClient(hc)
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
	for _, release := range releases {
		if release.TagName == nil {
			continue
		}
		for _, ignored := range ignoreVersions {
			if *release.TagName == ignored {
				continue
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

func getLatestCoreOS(ctx context.Context, ignoreVersions []string) (*neco.CoreOSImage, error) {
	resp, err := http.Get(coreOSFeed)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get CoreOS feed. status = %d", resp.StatusCode)
	}

	var feed map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&feed)
	if err != nil {
		return nil, err
	}

	versions := make([]*version.Version, 0, len(feed))
	for k := range feed {
		v, err := version.NewVersion(k)
		if err != nil {
			log.Error("bad version", map[string]interface{}{
				"version": k,
			})
			return nil, err
		}
		for _, ignored := range ignoreVersions {
			if k == ignored {
				continue
			}
		}
		versions = append(versions, v)
	}
	sort.Sort(sort.Reverse(version.Collection(versions)))
	return &neco.CoreOSImage{
		Channel: "stable",
		Version: versions[0].Original(),
	}, nil
}
