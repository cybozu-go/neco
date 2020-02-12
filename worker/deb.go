package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	"github.com/google/go-github/v18/github"
)

// InstallDebianPackage installs a debian package
// client uses for downloading a debian package.
// ghClient uses for getting download URL by GitHub API.
func InstallDebianPackage(ctx context.Context, client *http.Client, ghClient *http.Client, pkg *neco.DebianPackage, background bool) error {
	gh := neco.NewGitHubClient(ghClient)

	downloadURL, err := GetGithubDownloadURL(ctx, gh, pkg)
	if err != nil {
		return err
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	command := []string{"sh", "-c", "dpkg -i " + f.Name() + " && rm " + f.Name()}
	if background {
		command = append([]string{"systemd-run", "-q"}, command...)
	}
	return well.CommandContext(context.Background(), command[0], command[1:]...).Run()
}

func installLocalPackage(ctx context.Context, pkg *neco.DebianPackage) error {
	debVersion := pkg.Release[len("release-"):]
	deb := fmt.Sprintf("/tmp/%s_%s_amd64.deb", pkg.Name, debVersion)
	return well.CommandContext(context.Background(), "systemd-run", "-q", "dpkg", "-i", deb).Run()
}

// GetGithubDownloadURL returns URL of specified Debian package hosted in GitHub releases.
func GetGithubDownloadURL(ctx context.Context, gh *github.Client, pkg *neco.DebianPackage) (string, error) {
	releases, err := listGithubReleases(ctx, gh, pkg)
	if err != nil {
		return "", err
	}
	var release *github.RepositoryRelease
	for _, r := range releases {
		if r.TagName == nil {
			continue
		}
		if *r.TagName == pkg.Release {
			release = r
			break
		}
	}
	if release == nil {
		return "", fmt.Errorf("no such release: %s@%s", pkg.Repository, pkg.Release)
	}

	var asset *github.ReleaseAsset
	for _, a := range release.Assets {
		if strings.HasSuffix(*a.Name, ".deb") && strings.HasPrefix(*a.Name, "neco_") {
			asset = &a
			break
		}
	}
	if asset == nil {
		return "", errors.New("neco debian package not found")
	}

	if asset.BrowserDownloadURL == nil {
		return "", fmt.Errorf("asset browser-download-url is empty: %s@%s", pkg.Repository, pkg.Release)
	}

	return *asset.BrowserDownloadURL, nil
}

func listGithubReleases(ctx context.Context, gh *github.Client, pkg *neco.DebianPackage) ([]*github.RepositoryRelease, error) {
	opt := &github.ListOptions{PerPage: 100}
	var allReleases []*github.RepositoryRelease
	for {
		release, resp, err := gh.Repositories.ListReleases(ctx, pkg.Owner, pkg.Repository, opt)
		if err != nil {
			return nil, err
		}
		allReleases = append(allReleases, release...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allReleases, nil
}
