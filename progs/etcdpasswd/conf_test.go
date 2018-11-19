package etcdpasswd

import (
	"bytes"
	"testing"
)

func TestGenerateConf(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	err := GenerateConf(buf, []int{0, 1, 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(buf.String())
}
