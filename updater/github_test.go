package updater

import (
	"context"
	"testing"

	"github.com/cybozu-go/neco"
)

// These tests use https://github.com/neco-test/neco-ci

func testGetLatestReleaseTag(t *testing.T) {
	ghClient := neco.NewDefaultGitHubClient()
	c := NewReleaseClient("neco-test", "neco-ci", ghClient)
	ver, err := c.GetLatestReleaseTag(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ver != "2022.03.03-0303" {
		t.Errorf("unexpected version: %s", ver)
	}
}

func testGetLatestPublishedTag(t *testing.T) {
	ghClient := neco.NewDefaultGitHubClient()
	c := NewReleaseClient("neco-test", "neco-ci", ghClient)
	ver, err := c.GetLatestPublishedTag(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ver != "2022.04.04-0404" {
		t.Errorf("unexpected version: %s", ver)
	}

	c.SetTagPrefix("test-")
	ver, err = c.GetLatestPublishedTag(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ver != "2022.06.06-0606" {
		t.Errorf("unexpected version: %s", ver)
	}
}

func TestGitHub(t *testing.T) {
	t.Run("GetLatestReleaseTag", testGetLatestReleaseTag)
	t.Run("GetLatestPublishedTag", testGetLatestPublishedTag)
}
