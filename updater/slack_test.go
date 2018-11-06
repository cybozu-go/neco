package updater

import (
	"context"
	"testing"
	"time"
)

func testPostWebHook(t *testing.T) {
	t.Skip()

	ctx := context.Background()

	url := "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX"

	attachments := []Attachment{
		{
			Color:      ColorGood,
			AuthorName: "Boot server updater",
			Title:      "Update completed successfully",
			Text:       "Updating on boot servers are completed successfully :tada: :tada: :tada:",
			Fields: []AttachmentField{
				{Title: "Version", Value: "1.0.0", Short: true},
				{Title: "Servers", Value: "[1, 2, 3]", Short: true},
				{Title: "Reason", Value: "etcd servers are dead", Short: true},
				{Title: "Started at", Value: time.Now().Format(time.RFC3339), Short: true},
			},
		},
		{
			Color:      ColorDanger,
			AuthorName: "Boot server updater",
			Title:      "Update failed",
			Text:       "Updating on boot servers failed :crying_cat_face: .  Please repair updating on the boot servers manually",
			Fields: []AttachmentField{
				{Title: "Version", Value: "1.0.0", Short: true},
				{Title: "Servers", Value: "[1, 2, 3]", Short: true},
				{Title: "Reason", Value: "etcd servers are dead", Short: true},
				{Title: "Started at", Value: time.Now().Format(time.RFC3339), Short: true},
			},
		},
	}

	c := SlackClient{URL: url}
	err := c.PostWebHook(ctx, Payload{Attachments: attachments})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSlackClient(t *testing.T) {

	t.Run("PostWebHook", testPostWebHook)
}
