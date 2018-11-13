package ext

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
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

// NotifySucceeded notifies to slack that update was successful.
func (c SlackClient) NotifySucceeded(ctx context.Context, req neco.UpdateRequest) error {
	att := Attachment{
		Color:      ColorGood,
		AuthorName: "Boot server updater",
		Title:      "Update completed successfully",
		Text:       "Updating on boot servers are completed successfully :tada: :tada: :tada:",
		Fields: []AttachmentField{
			{Title: "Version", Value: req.Version, Short: true},
			{Title: "Servers", Value: fmt.Sprintf("%v", req.Servers), Short: true},
			{Title: "Started at", Value: req.StartedAt.Format(time.RFC3339), Short: true},
		},
	}
	payload := Payload{Attachments: []Attachment{att}}
	return c.PostWebHook(ctx, payload)
}

// NotifyServerFailure notifies to slack that update was failure.
func (c SlackClient) NotifyServerFailure(ctx context.Context, req neco.UpdateRequest, message string) error {
	att := Attachment{
		Color:      ColorDanger,
		AuthorName: "Boot server updater",
		Title:      "Failed to update boot servers",
		Text:       "Failed to update boot servers due to some worker return(s) error :crying_cat_face:.  Please fix it manually.",
		Fields: []AttachmentField{
			{Title: "Version", Value: req.Version, Short: true},
			{Title: "Servers", Value: fmt.Sprintf("%v", req.Servers), Short: true},
			{Title: "Started at", Value: req.StartedAt.Format(time.RFC3339), Short: true},
			{Title: "Reason", Value: message, Short: true},
		},
	}
	payload := Payload{Attachments: []Attachment{att}}
	return c.PostWebHook(ctx, payload)
}

// NotifyTimeout notifies to slack that update has timed out.
func (c SlackClient) NotifyTimeout(ctx context.Context, req neco.UpdateRequest) error {
	att := Attachment{
		Color:      ColorDanger,
		AuthorName: "Boot server updater",
		Title:      "Update failed on the boot servers",
		Text:       "Failed to update boot servers due to timed-out from worker updates :crying_cat_face:.  Please fix it manually.",
		Fields: []AttachmentField{
			{Title: "Version", Value: req.Version, Short: true},
			{Title: "Servers", Value: fmt.Sprintf("%v", req.Servers), Short: true},
			{Title: "Started at", Value: req.StartedAt.Format(time.RFC3339), Short: true},
		},
	}
	payload := Payload{Attachments: []Attachment{att}}
	return c.PostWebHook(ctx, payload)
}
