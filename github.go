package neco

import (
	"context"
	"net/http"
	"os"

	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
)

// NewGitHubClient returns a properly configured *github.Client.
func NewGitHubClient(c *http.Client) *github.Client {
	gh := github.NewClient(c)
	gh.UserAgent = NecoUserAgent
	return gh
}

// NewDefaultGitHubClient reads GitHub personal access token from GITHUB_TOKEN environment variable.
// And creates a properly configured *github.Client.
func NewDefaultGitHubClient() *github.Client {
	var hc *http.Client
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		src := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		hc = oauth2.NewClient(context.Background(), src)
	}
	return NewGitHubClient(hc)
}
