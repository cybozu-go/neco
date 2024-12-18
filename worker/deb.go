package worker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	"github.com/google/go-github/v50/github"
)

// InstallDebianPackage installs a debian package
// client is used to download a debian package. If token is not empty, it is used as a bearer token.
// ghClient uses for getting download URL by GitHub API.
// envvalues uses for giving environment values to systemd-run command when background is true.
func InstallDebianPackage(ctx context.Context, client *http.Client, token string, ghClient *github.Client, pkg *neco.DebianPackage, background bool, envValues map[string]string) error {
	downloadURL, err := GetGitHubDownloadURL(ctx, ghClient, pkg)
	if err != nil {
		return err
	}

	log.Info("installing debian package...", map[string]interface{}{
		"package": pkg.Name,
		"release": pkg.Release,
		"url":     downloadURL,
	})
	var data []byte
	const retryCount = 10
	err = neco.RetryWithSleep(ctx, retryCount, 10*time.Second,
		func(ctx context.Context) error {
			req, err := http.NewRequest("GET", downloadURL, nil)
			if err != nil {
				return err
			}
			req.Header.Add("Accept", "application/octet-stream")
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
				return fmt.Errorf("status code %d is not a success", resp.StatusCode)
			}

			data, err = io.ReadAll(resp.Body)
			return err
		},
		func(err error) {
			log.Warn("deb: failed to download debian package", map[string]interface{}{
				log.FnError:   err,
				"downloadURL": downloadURL,
			})
		},
	)
	if err != nil {
		return err
	}

	f, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	command := []string{"sh", "-c", "dpkg -i " + f.Name() + " && rm " + f.Name()}
	if background {
		systemdCommand := []string{"systemd-run", "-q", "--wait"}
		if envValues != nil {
			for k, v := range envValues {
				systemdCommand = append(systemdCommand, []string{"-p", fmt.Sprintf("Environment=%s=%s", k, v)}...)
			}
			command = append(systemdCommand, command...)
		}
	}

	return well.CommandContext(context.Background(), command[0], command[1:]...).Run()
}

func installLocalPackage(ctx context.Context, pkg *neco.DebianPackage, envValues map[string]string) error {
	debVersion := pkg.Release[len("release-"):]
	deb := fmt.Sprintf("/tmp/%s_%s_amd64.deb", pkg.Name, debVersion)
	command := []string{"systemd-run", "-q", "--wait"}
	if envValues != nil {
		envs := make([]string, 0, len(envValues)*2)
		for k, v := range envValues {
			envs = append(envs, []string{"-p", fmt.Sprintf("Environment=%s=%s", k, v)}...)
		}
		command = append(command, envs...)
	}
	command = append(command, []string{"dpkg", "-i", deb}...)
	return well.CommandContext(context.Background(), command[0], command[1:]...).Run()
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
	if asset.URL == nil {
		return "", fmt.Errorf("asset url is empty: %s@%s", pkg.Repository, pkg.Release)
	}

	return *asset.URL, nil
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
