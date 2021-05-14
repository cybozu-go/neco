package worker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	"github.com/google/go-github/v35/github"
)

// InstallDebianPackage installs a debian package
// client uses for downloading a debian package.
// ghClient uses for getting download URL by GitHub API.
func InstallDebianPackage(ctx context.Context, client *http.Client, ghClient *http.Client, pkg *neco.DebianPackage, background bool) error {
	gh := neco.NewGitHubClient(ghClient)

	downloadURL, err := GetGitHubDownloadURL(ctx, gh, pkg)
	if err != nil {
		return err
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.CreateTemp("", "")
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

// GetGitHubDownloadURL returns URL of specified Debian package hosted in GitHub releases.
func GetGitHubDownloadURL(ctx context.Context, gh *github.Client, pkg *neco.DebianPackage) (string, error) {
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

	asset := findDebAsset(release.Assets, pkg.Name)
	if asset == nil {
		return "", fmt.Errorf("debian package not found: %s@%s", pkg.Repository, pkg.Release)
	}
	if asset.BrowserDownloadURL == nil {
		return "", fmt.Errorf("asset browser-download-url is empty: %s@%s", pkg.Repository, pkg.Release)
	}

	return *asset.BrowserDownloadURL, nil
}

func findDebAsset(assets []*github.ReleaseAsset, name string) *github.ReleaseAsset {
	filePattern := regexp.MustCompile(fmt.Sprintf(`^%s_.*\.deb$`, name))
	for _, a := range assets {
		name := a.Name
		if name == nil {
			continue
		}
		if filePattern.MatchString(*name) {
			return a
		}
	}
	return nil
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
