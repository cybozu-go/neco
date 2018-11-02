package neco

import (
	"context"
	"testing"
)

func testFetchContainer(t *testing.T) {
	err := FetchContainer(context.Background(), "vault")
	if err != nil {
		t.Fatal(err)
	}
}

func testRunContainer(t *testing.T) {
	err := RunContainer(context.Background(), "vault",
		[]Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/vault/install-tools"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRkt(t *testing.T) {
	t.Skip()
	t.Run("FetchContainer", testFetchContainer)
	t.Run("RunContainer", testRunContainer)
}
