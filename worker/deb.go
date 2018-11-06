package worker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	"github.com/google/go-github/v18/github"
)

// GetDebianVersion returns debian package version.
// If "neco" package is not installed, this returns ("", nil).
func GetDebianVersion(pkg string) (string, error) {
	if exec.Command("dpkg", "-s", pkg).Run() != nil {
		return "", nil
	}

	data, err := well.CommandContext(context.Background(),
		"dpkg-query", "--showformat=${Version}", "-W", pkg).Output()
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// InstallDebianPackage installs a debian package
func InstallDebianPackage(ctx context.Context, client *http.Client, pkg *neco.DebianPackage) error {
	gh := github.NewClient(client)

	releases, err := listGithubReleases(ctx, gh, pkg)
	if err != nil {
		return err
	}
	var release *github.RepositoryRelease
	for _, release = range releases {
		if release.TagName != nil && *release.TagName == pkg.Release {
			break
		}
	}
	if release == nil {
		return fmt.Errorf("no such release: %s@%s", pkg.Repository, pkg.Release)
	}

	if len(release.Assets) != 1 {
		return fmt.Errorf("no asset in release: %s@%s", pkg.Repository, pkg.Release)
	}

	asset := release.Assets[0]
	if asset.BrowserDownloadURL == nil {
		return fmt.Errorf("asset browser-download-url is empty: %s@%s", pkg.Repository, pkg.Release)
	}

	resp, err := client.Get(*asset.BrowserDownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	return well.CommandContext(ctx, "dpkg", "-i", f.Name()).Run()
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
