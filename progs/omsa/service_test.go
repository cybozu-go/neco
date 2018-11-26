package omsa

import (
	"bytes"
	"testing"
)

func TestGenerateService(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	err := GenerateService(buf)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(buf.String())
}
