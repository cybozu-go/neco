package updater

import (
	"context"
	"testing"
)

func testGetLatestReleaseTag(t *testing.T) {
	t.Skip()

	c := ReleaseClient{owner: "kubernetes", repo: "kubernetes"}
	ver, err := c.GetLatestReleaseTag(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("version = %s", ver)
}

func testGetLatestPublishedTag(t *testing.T) {
	t.Skip()

	c := ReleaseClient{owner: "kubernetes", repo: "kubernetes"}
	ver, err := c.GetLatestPublishedTag(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// The method returns `v1.0.5' since Kubernetes publish pre-release
	// with tag as `v1.13.0-alpha.3'.  The version is parsed as `alpha.3'
	// and it is skipped in the method.
	t.Logf("version = %s", ver)
}

func TestGitHub(t *testing.T) {
	t.Run("GetLatestReleaseTag", testGetLatestReleaseTag)
	t.Run("GetLatestPublishedTag", testGetLatestPublishedTag)
}
