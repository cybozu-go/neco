package neco

import "testing"

func TestGetDebianVersion(t *testing.T) {
	t.Parallel()

	v, err := GetDebianVersion("bash")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(v)

	v, err = GetDebianVersion("no-such-package")
	if err != nil {
		t.Fatal(err)
	}
	if v != "" {
		t.Error("no-such-package version should be empty")
	}
}
