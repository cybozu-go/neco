package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

func (zh *ZenHubClient) request(ctx context.Context, query string, vars map[string]interface{}) ([]byte, error) {
	greq := []graphQLRequest{{Query: query, Variables: vars}}
	data, err := json.Marshal(greq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, zenhubGraphQLV1Endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
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

	var gresp []graphQLResponse
	err = json.NewDecoder(resp.Body).Decode(&gresp)
	if err != nil {
		return nil, err
	}

	if len(gresp) != 1 {
		return nil, fmt.Errorf("response should have only 1 item, but got %d", len(gresp))
	}

	r := gresp[0]
	if len(r.Errors) > 0 {
		return nil, errors.New(r.Errors[0].Message)
	}
	return []byte(r.Data), nil
}

// Queries are copied from the value showen on the Network tab on Chrome
// Developer tool when manually connecting a PR with an issue.

// Connect connect a pull request with an issue.
func (zh *ZenHubClient) Connect(ctx context.Context, issueID, pullRequestID string) error {
	query := "mutation CreateIssuePrConnection($input: CreateIssuePrConnectionInput!) {\n  createIssuePrConnection(input: $input) {\n    issue {\n      id\n      __typename\n    }\n    __typename\n  }\n}\n"
	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"issueId":       issueID,
			"pullRequestId": pullRequestID,
		},
	}

	_, err := zh.request(ctx, query, vars)
	if err != nil {
		return err
	}
	return nil
}

// GetIssueID gets a zenhub internal ID for either PR or issue.
func (zh *ZenHubClient) GetIssueID(ctx context.Context, repositoryGhID, issueGhNumber int) (string, error) {
	query := fmt.Sprintf(
		"query IssueByInfo {\n  issueByInfo(repositoryGhId: %d, issueNumber: %d) {\n    id\n    __typename\n  }\n}\n",
		repositoryGhID,
		issueGhNumber,
	)
	vars := map[string]interface{}{}

	data, err := zh.request(ctx, query, vars)
	if err != nil {
		return "", err
	}

	var resp struct {
		IssueByInfo struct {
			ID string `json:"id"`
		} `json:"issueByInfo"`
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return "", err
	}

	return resp.IssueByInfo.ID, nil
}
