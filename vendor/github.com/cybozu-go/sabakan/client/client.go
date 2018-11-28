package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/user"
	"path"

	"github.com/cybozu-go/log"
)

// Client is a sabakan client
type Client struct {
	url      *url.URL
	http     *http.Client
	username string
}

// NewClient returns new client
func NewClient(endpoint string, http *http.Client) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	username := user.Username

	client := &Client{
		url:      u,
		http:     http,
		username: username,
	}
	return client, nil
}

// newRequest creates a new http.Request whose context is set to ctx.
// path will be prefixed by "/api/v1".
func (c *Client) newRequest(ctx context.Context, method, p string, body io.Reader) *http.Request {
	u := *c.url
	u.Path = path.Join(u.Path, "/api/v1", p)
	r, _ := http.NewRequest(method, u.String(), body)
	r.Header.Set("X-Sabakan-User", c.username)
	return r.WithContext(ctx)
}

// do calls http.Client.Do and processes errors.
// This returns non-nil *http.Response only when the server returns 2xx status code.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	switch {
	case 200 <= resp.StatusCode && resp.StatusCode < 400:
	case 400 <= resp.StatusCode && resp.StatusCode < 600:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()

		var msg map[string]interface{}
		err = json.Unmarshal(body, &msg)
		if err != nil {
			return nil, &httpError{code: resp.StatusCode, reason: string(body)}
		}
		reason := fmt.Sprintf("%s", msg[log.FnError])
		return nil, &httpError{code: resp.StatusCode, reason: reason}

	}
	return resp, nil
}

func (c *Client) getJSON(ctx context.Context, p string, params map[string]string, data interface{}) error {
	req := c.newRequest(ctx, "GET", p, nil)
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) getBytes(ctx context.Context, p string) ([]byte, error) {
	req := c.newRequest(ctx, "GET", p, nil)
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *Client) sendRequestWithJSON(ctx context.Context, method, p string, data interface{}) error {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(data)
	if err != nil {
		return err
	}

	req := c.newRequest(ctx, method, p, b)
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (c *Client) sendRequest(ctx context.Context, method, p string, r io.Reader) error {
	req := c.newRequest(ctx, method, p, r)
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
