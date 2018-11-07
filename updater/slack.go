package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cybozu-go/log"
)

// Reserved colors in Slack API
const (
	ColorGood    = "good"
	ColorWarning = "warning"
	ColorDanger  = "danger"
)

// Payload represents a slack payload
type Payload struct {
	Attachments []Attachment `json:"attachments"`
}

// AttachmentField represents fields in a Attachment
type AttachmentField struct {
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
	Short bool   `json:"short,omitempty"`
}

// Attachment represents an attachment in the slack payload
type Attachment struct {
	Fallback   string            `json:"fallback,omitempty"`
	Color      string            `json:"color,omitempty"`
	Pretext    string            `json:"pretext,omitempty"`
	AuthorName string            `json:"author_name,omitempty"`
	AuthorLink string            `json:"author_link,omitempty"`
	AuthorIcon string            `json:"author_icon,omitempty"`
	Title      string            `json:"title,omitempty"`
	TitleLink  string            `json:"title_link,omitempty"`
	Text       string            `json:"text,omitempty"`
	Fields     []AttachmentField `json:"fields,omitempty"`
	ImageURL   string            `json:"image_url,omitempty"`
	ThumbURL   string            `json:"thumb_url,omitempty"`
	Footer     string            `json:"footer,omitempty"`
	FooterIcon string            `json:"footer_icon,omitempty"`
	Timestamp  time.Time         `json:"ts,omitempty"`
}

// SlackClient is a slack client
type SlackClient struct {
	URL  string
	HTTP *http.Client
}

// PostWebHook posts a payload to slack
func (c SlackClient) PostWebHook(ctx context.Context, payload Payload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	httpc := c.HTTP
	if httpc == nil {
		httpc = http.DefaultClient
	}

	_, err = httpc.Do(req)
	if err != nil {
		log.Warn("Failed to send slack notification", map[string]interface{}{
			"content-length": len(body),
			"error":          err,
		})
	}
	return err
}
