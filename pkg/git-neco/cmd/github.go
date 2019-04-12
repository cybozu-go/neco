package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/oauth2"
)

const githubAPIv4Endpoint = "https://api.github.com/graphql"

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

type GitHubUser struct {
	ID    string
	Login string
}

// GitHubRepoName represents a repository hosted on GitHub.
type GitHubRepoName struct {
	Owner string
	Name  string
}

type GitHubRepoID struct {
	ID     string
	Number int
}

type PullRequest struct {
	ID        string
	Number    int
	Permalink string
}

func ExtractGitHubRepoName(url string) *GitHubRepoName {
	// Extract repository name from SCP-like address
	// Eg, git@github.com:user/repo.git -> user/repo.git
	reg := regexp.MustCompile(`^([a-zA-Z0-9_]+)@([a-zA-Z0-9._-]+):(.*)$`)
	if m := reg.FindStringSubmatch(url); m != nil {
		url = m[3]
	}

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return nil
	}
	return &GitHubRepoName{
		Owner: parts[len(parts)-2],
		Name:  strings.Split(parts[len(parts)-1], ".")[0],
	}
}

// NewGitHubClient creates GitHubClient
func NewGitHubClient(ctx context.Context, token string) GitHubClient {
	endpoint, _ := url.Parse(githubAPIv4Endpoint)
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

// GetRepository returns global node ID of the repo.
func (gh GitHubClient) GetRepository(ctx context.Context, repo *GitHubRepoName) (*GitHubRepoID, error) {
	query := `query getRepository($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    id,
    databaseId
  }
}`
	vars := map[string]interface{}{
		"owner": repo.Owner,
		"name":  repo.Name,
	}

	data, err := gh.request(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Repository struct {
			ID         string `json:"id"`
			DatabaseID int    `json:"databaseId"`
		} `json:"repository"`
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}
	return &GitHubRepoID{
		ID:     resp.Repository.ID,
		Number: resp.Repository.DatabaseID,
	}, nil
}

// GetViewer returns global node ID and Login ID of the login user.
func (gh GitHubClient) GetViewer(ctx context.Context) (*GitHubUser, error) {
	query := `query {
  viewer {
    id,
	login
  }
}`
	vars := map[string]interface{}{}
	data, err := gh.request(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Viewer struct {
			ID    string `json:"id"`
			Login string `json:"login"`
		} `json:"viewer"`
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}

	return &GitHubUser{
		ID:    resp.Viewer.ID,
		Login: resp.Viewer.Login,
	}, nil
}

// GetDraftPR returns a draft pull request in a repository for a branch.
// If no pull request is found, this returns ("", nil).
func (gh GitHubClient) GetDraftPR(ctx context.Context, repo *GitHubRepoName, branch string) (string, error) {
	query := `query getDraftPR($owner: String!, $name: String!, $headRef: String!) {
  repository(owner: $owner, name: $name) {
    pullRequests(headRefName: $headRef, first: 100) {
      nodes {
        id,
        isDraft,
        state,
      }
    }
  }
}`
	vars := map[string]interface{}{
		"owner":   repo.Owner,
		"name":    repo.Name,
		"headRef": branch,
	}

	data, err := gh.request(ctx, query, vars)
	if err != nil {
		return "", err
	}

	var resp struct {
		Repository struct {
			PullRequests struct {
				Nodes []struct {
					ID      string `json:"id"`
					IsDraft bool   `json:"isDraft"`
					State   string `json:"state"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return "", err
	}

	var id string
	for _, pr := range resp.Repository.PullRequests.Nodes {
		if !pr.IsDraft {
			continue
		}
		if pr.State != "OPEN" {
			continue
		}
		id = pr.ID
		break
	}
	return id, nil
}

// MarkDraftReadyForReview marks a draft pull request ready for review.
func (gh GitHubClient) MarkDraftReadyForReview(ctx context.Context, id string) error {
	query := `mutation mark($input: MarkPullRequestReadyForReviewInput!) {
  markPullRequestReadyForReview(input: $input) {
    clientMutationId
  }
}`

	input := struct {
		ID string `json:"pullRequestId"`
	}{
		ID: id,
	}
	vars := map[string]interface{}{
		"input": input,
	}

	_, err := gh.request(ctx, query, vars)
	return err
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
// When succeeds, it returns this:
//  - a global node ID of the new PR
//  - a number of the new PR
//  - a permalink to the new PR
func (gh GitHubClient) CreatePullRequest(ctx context.Context, repo, base, head, title, body string, draft bool) (*PullRequest, error) {
	query := `mutation createPR($input: CreatePullRequestInput!) {
  createPullRequest(input: $input) {
    pullRequest {
      id,
      number,
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
		return nil, err
	}

	var resp struct {
		CreatePullRequest struct {
			PullRequest struct {
				ID        string `json:"id"`
				Number    int    `json:"number"`
				Permalink string `json:"permalink"`
			} `json:"pullRequest"`
		} `json:"createPullRequest"`
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}

	return &PullRequest{
		ID:        resp.CreatePullRequest.PullRequest.ID,
		Number:    resp.CreatePullRequest.PullRequest.Number,
		Permalink: resp.CreatePullRequest.PullRequest.Permalink,
	}, nil
}

type addAssigneeInput struct {
	AssigneeID    string `json:"assigneeIds"`
	PullRequestID string `json:"assignableId"`
}

// AddAssigneeToPullRequest add a assignee to a pull request.
func (gh GitHubClient) AddAssigneeToPullRequest(ctx context.Context, userID, prID string) error {
	query := `mutation addAssignee($input: AddAssigneesToAssignableInput!) {
  addAssigneesToAssignable(input: $input) {
    clientMutationId
  }
}`
	input := addAssigneeInput{
		AssigneeID:    userID,
		PullRequestID: prID,
	}
	vars := map[string]interface{}{
		"input": input,
	}

	_, err := gh.request(ctx, query, vars)
	return err
}
