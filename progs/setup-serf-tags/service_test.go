package setupserftags

import (
	"bytes"
	"testing"
)

func TestGenerateService(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	err := GenerateScript(buf, "0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(buf.String())
}
