package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

const zenhubAPIv4Endpoint = "https://api.zenhub.io/v4"

func getZenHubURL(addPath string) *url.URL {
	u, _ := url.Parse(zenhubAPIv4Endpoint)
	u.Path = path.Join(u.Path, addPath)
	return u
}

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
	req.Header.Set("X-Authentication-Token", zh.token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if 200 <= resp.StatusCode && resp.StatusCode <= 299 {
		return io.ReadAll(resp.Body)
	}

	// Error handling
	var errResp struct {
		Message string `json:"message"`
	}
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	if err != nil {
		return nil, err
	}
	return nil, errors.New(errResp.Message)
}

// Connect connect a pull request with an issue.
func (zh *ZenHubClient) Connect(ctx context.Context, issueRepo int, issue int, prRepo int, pr int) error {
	u := getZenHubURL(fmt.Sprintf("repositories/%d/connection", issueRepo))

	v := url.Values{}
	v.Set("issue_number", strconv.Itoa(issue))
	v.Add("connected_repo_id", strconv.Itoa(prRepo))
	v.Add("connected_issue_number", strconv.Itoa(pr))

	_, err := zh.request(ctx, http.MethodPost, u.String(), v.Encode())
	if err != nil {
		return err
	}
	return nil
}
