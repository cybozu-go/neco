package neco

import (
	"net/http"

	"github.com/google/go-github/v41/github"
)

// NewGitHubClient returns a properly configured *github.Client.
func NewGitHubClient(c *http.Client) *github.Client {
	gh := github.NewClient(c)
	gh.UserAgent = NecoUserAgent
	return gh
}
