package etcd

import (
	"bytes"
	"context"
	"testing"
)

func TestGenerateService(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	err := GenerateService(context.Background(), buf)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(buf.String())
}
