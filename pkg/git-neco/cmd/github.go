package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

var (
	githubPreviewHeaders = []string{
		"application/vnd.github.ocelot-preview+json",
		"application/vnd.github.shadow-cat-preview+json",
	}
)

// GitHubClient implements a partial GitHub GraphQL API v4.
type GitHubClient struct {
	endpoint *url.URL
	hc       *http.Client
}

// NewGitHubClient creates GitHubClient
func NewGitHubClient(ctx context.Context, endpoint *url.URL, token string) GitHubClient {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	return GitHubClient{
		endpoint: endpoint,
		hc:       oauth2.NewClient(ctx, src),
	}
}

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

func (gh GitHubClient) request(ctx context.Context, query string, vars map[string]interface{}) ([]byte, error) {
	greq := graphQLRequest{Query: query, Variables: vars}
	data, err := json.Marshal(greq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", gh.endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", strings.Join(githubPreviewHeaders, ","))

	resp, err := gh.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gresp graphQLResponse
	err = json.NewDecoder(resp.Body).Decode(&gresp)
	if err != nil {
		return nil, err
	}

	if len(gresp.Errors) > 0 {
		return nil, errors.New(gresp.Errors[0].Message)
	}
	return []byte(gresp.Data), nil
}

// Repository returns global node ID of the repo.
func (gh GitHubClient) Repository(ctx context.Context, repo GitHubRepo) (string, error) {
	query := `query getRepositoryID($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    id
  }
}`
	vars := map[string]interface{}{
		"owner": repo.Owner,
		"name":  repo.Name,
	}

	data, err := gh.request(ctx, query, vars)
	if err != nil {
		return "", err
	}

	var resp struct {
		Repository struct {
			ID string `json:"id"`
		} `json:"repository"`
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return "", err
	}
	return resp.Repository.ID, nil
}

type pullRequestInput struct {
	RepositoryID string `json:"repositoryId"`
	BaseRefName  string `json:"baseRefName"`
	HeadRefName  string `json:"headRefName"`
	Title        string `json:"title"`
	Body         string `json:"body,omitempty"`
	Draft        bool   `json:"draft,omitempty"`
}

// CreatePullRequest creates a new pull request to merge head into base.
// title must not be empty.
// When succeeds, it returns a permalink to the new PR.
func (gh GitHubClient) CreatePullRequest(ctx context.Context, repo, base, head, title, body string, draft bool) (string, error) {
	query := `mutation createPR($input: CreatePullRequestInput!) {
  createPullRequest(input: $input) {
    pullRequest {
		permalink
	}
  }
}`
	input := pullRequestInput{
		RepositoryID: repo,
		BaseRefName:  base,
		HeadRefName:  head,
		Title:        title,
		Body:         body,
		Draft:        draft,
	}
	vars := map[string]interface{}{
		"input": input,
	}

	data, err := gh.request(ctx, query, vars)
	if err != nil {
		return "", err
	}

	var resp struct {
		CreatePullRequest struct {
			PullRequest struct {
				Permalink string `json:"permalink"`
			} `json:"pullRequest"`
		} `json:"createPullRequest"`
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return "", err
	}

	return resp.CreatePullRequest.PullRequest.Permalink, nil
}
