package cmd

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

func sanityCheck() error {
	return exec.Command("git", "status").Run()
}

func git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitOutput(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func currentBranch() (string, error) {
	data, err := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	br := strings.TrimSpace(string(data))
	if br == "HEAD" {
		return "", errors.New("not in a local branch")
	}
	return br, nil
}

func checkUncommittedFiles() (bool, error) {
	data, err := gitOutput("status", "-s")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(data)) == "", nil
}

func firstUnmerged() (hash string, summary string, body string, err error) {
	var data []byte
	data, err = gitOutput("log", "HEAD", "--not", "origin/master", "--format=%H", "--reverse")
	if err != nil {
		return
	}
	commits := strings.Fields(string(data))
	if len(commits) == 0 {
		err = errors.New("no commits to be pushed")
		return
	}

	hash = commits[0]

	data, err = gitOutput("show", "--format=%s%n%b", "-s", hash)
	if err != nil {
		return
	}

	fields := strings.SplitN(string(data), "\n", 2)
	return hash, fields[0], strings.TrimSpace(fields[1]), nil
}

// GitHubRepo represents a repository hosted on GitHub.
type GitHubRepo struct {
	Owner string
	Name  string
}

// CurrentRepo returns GitHubRepo for the current git worktree.
func CurrentRepo() (*GitHubRepo, error) {
	data, err := gitOutput("remote", "get-url", "origin")
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.TrimSpace(string(data)), "/")
	if len(parts) < 2 {
		return nil, errors.New("bad remote URL: " + string(data))
	}
	return &GitHubRepo{
		Owner: parts[len(parts)-2],
		Name:  strings.Split(parts[len(parts)-1], ".")[0],
	}, nil
}

// GetEndpointInfo returns an endpoint and a token for the current git worktree.
func GetEndpointInfo() (*url.URL, string, error) {
	repo, err := CurrentRepo()
	if err != nil {
		return nil, "", err
	}
	endpoint, _ := url.Parse("https://api.github.com/graphql")
	token := config.GithubToken
	switch repo.Name {
	case "wiki":
		u, err := url.Parse(config.GheURL)
		if err != nil {
			return nil, "", err
		}
		u.Path = "/api/graphql"
		endpoint = u
		token = config.GheToken
	}
	return endpoint, token, nil
}
