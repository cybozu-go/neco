package neco

import (
	"context"
	"testing"
)

func testFetchContainer(t *testing.T) {
	fullname, err := ContainerFullName("vault")
	if err != nil {
		t.Fatal(err)
	}
	err = FetchContainer(context.Background(), fullname, nil)
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
