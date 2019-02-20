package client

import (
	"context"
	"io"
	"time"
)

// LogsGet retrieves audit logs.
func (c *Client) LogsGet(ctx context.Context, since, until time.Time, w io.Writer) error {
	req := c.newRequest(ctx, "GET", "logs", nil)
	q := req.URL.Query()
	if !since.IsZero() {
		q.Set("since", since.UTC().Format("20060102"))
	}
	if !until.IsZero() {
		q.Set("until", until.UTC().Format("20060102"))
	}
	req.URL.RawQuery = q.Encode()

	resp, status := c.do(req)
	if status != nil {
		return status
	}
	defer resp.Body.Close()

	_, err := io.Copy(w, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
