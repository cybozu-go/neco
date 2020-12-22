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
	ColorInfo    = "#439FE0"
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
}

// SlackClient is a slack client
type SlackClient struct {
	URL     string
	HTTP    *http.Client
	Cluster string
}

// PostWebHook posts a payload to slack
func (c SlackClient) PostWebHook(payload Payload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

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

// NotifyInfo sends a notification about the beginning of the update process
func (c SlackClient) NotifyInfo(req neco.UpdateRequest, message string) error {
	att := Attachment{
		Color:      ColorInfo,
		AuthorName: "Boot server updater",
		Title:      "Update begins",
		Text:       "neco-worker has started the updating process.",
		Fields: []AttachmentField{
			{Title: "Cluster", Value: c.Cluster, Short: true},
			{Title: "Version", Value: req.Version, Short: true},
			{Title: "Servers", Value: fmt.Sprintf("%v", req.Servers), Short: true},
			{Title: "Started at", Value: req.StartedAt.Format(time.RFC3339), Short: true},
			{Title: "Detail", Value: message, Short: false},
		},
	}
	payload := Payload{Attachments: []Attachment{att}}
	return c.PostWebHook(payload)
}

// NotifySucceeded sends a successful notification about the update process
func (c SlackClient) NotifySucceeded(req neco.UpdateRequest) error {
	att := Attachment{
		Color:      ColorGood,
		AuthorName: "Boot server updater",
		Title:      "Update completed successfully",
		Text:       "boot servers were updated successfully :tada: :tada: :tada:",
		Fields: []AttachmentField{
			{Title: "Cluster", Value: c.Cluster, Short: true},
			{Title: "Version", Value: req.Version, Short: true},
			{Title: "Servers", Value: fmt.Sprintf("%v", req.Servers), Short: true},
			{Title: "Started at", Value: req.StartedAt.Format(time.RFC3339), Short: true},
		},
	}
	payload := Payload{Attachments: []Attachment{att}}
	return c.PostWebHook(payload)
}

// NotifyFailure sends a failure notification about the update process
func (c SlackClient) NotifyFailure(req neco.UpdateRequest, message string) error {
	att := Attachment{
		Color:      ColorDanger,
		AuthorName: "Boot server updater",
		Title:      "Update failed",
		Text:       "there were some errors :crying_cat_face:.  Please fix it manually.",
		Fields: []AttachmentField{
			{Title: "Cluster", Value: c.Cluster, Short: true},
			{Title: "Version", Value: req.Version, Short: true},
			{Title: "Servers", Value: fmt.Sprintf("%v", req.Servers), Short: true},
			{Title: "Started at", Value: req.StartedAt.Format(time.RFC3339), Short: true},
			{Title: "Reason", Value: message, Short: false},
		},
	}
	payload := Payload{Attachments: []Attachment{att}}
	return c.PostWebHook(payload)
}
