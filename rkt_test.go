package neco

import (
	"context"
	"testing"
)

func TestFetchContainer(t *testing.T) {
	t.Skip()

	err := FetchContainer(context.Background(), "serf")
	if err != nil {
		t.Fatal(err)
	}
}
