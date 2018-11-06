package worker

import "testing"

func TestGetDebianVersion(t *testing.T) {
	t.Parallel()

	v, err := GetDebianVersion("bash")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(v)
}
