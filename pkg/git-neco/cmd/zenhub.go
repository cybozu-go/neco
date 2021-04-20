package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const zenhubGraphQLV1Endpoint = "https://api.zenhub.io/v1/graphql"

// ZenHubClient implements a partial ZenHub API.
type ZenHubClient struct {
	token string
}

// NewZenHubClient creates ZenHubClient.
func NewZenHubClient(token string) *ZenHubClient {
	return &ZenHubClient{
		token: token,
	}
}

func (zh *ZenHubClient) request(ctx context.Context, method string, url string, body string) ([]byte, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Method = "POST"
	req.Header.Set("X-Authentication-Token", zh.token)
	req.Header.Add("Content-Type", "application/json")

	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code should be 200, but got %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rl []responseItem
	err = json.Unmarshal(b, &rl)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, b)
	}

	for _, r := range rl {
		if len(r.Errors) == 0 {
			continue
		}
		return nil, fmt.Errorf("got response with errors: %s", string(b))
	}
	return b, nil
}

// Connect connect a pull request with an issue.
func (zh *ZenHubClient) Connect(ctx context.Context, issueID, pullRequestID string) error {
	body, err := json.Marshal(newPayload(issueID, pullRequestID))
	if err != nil {
		return err
	}

	_, err = zh.request(ctx, http.MethodPost, zenhubGraphQLV1Endpoint, string(body))
	if err != nil {
		return err
	}
	return nil
}

type payloadItem struct {
	OperationName string     `json:"operationName"`
	Query         string     `json:"query"`
	Variables     *variables `json:"variables"`
}

type variables struct {
	Input *zenHubInput `json:"input"`
}

type zenHubInput struct {
	IssueID       string `json:"issueId"`
	PullRequestID string `json:"pullRequestId"`
}

type responseItem struct {
	Errors []errorItem `json:"errors"`
}

type errorItem struct {
	Message string `json:"message"`
}

// This payload can be observed by connecting PR with issue on ZenHub WebUI and
// open Network tab of Chrome Developer Tool
func newPayload(issueID, pullRequestID string) []payloadItem {
	return []payloadItem{
		{
			OperationName: "CreateIssuePrConnection",
			Query:         "mutation CreateIssuePrConnection($input: CreateIssuePrConnectionInput!) {\n  createIssuePrConnection(input: $input) {\n    issue {\n      id\n      __typename\n    }\n    __typename\n  }\n}\n",
			Variables: &variables{
				&zenHubInput{
					IssueID:       issueID,
					PullRequestID: pullRequestID,
				},
			},
		},
	}
}
