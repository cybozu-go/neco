package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/cybozu-go/well"
)

var httpClient = &well.HTTPClient{
	Client: &http.Client{},
}

// Payload represents slack payload
type Payload struct {
	Channel   string `json:"channel,omitempty"`
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
	Text      string `json:"text,omitempty"`
}

// NotifySlack notifies to slack
func NotifySlack(ctx context.Context, url string, payload Payload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	_, err = httpClient.Do(req)
	return err
}
