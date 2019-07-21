package neco

import "testing"

func TestGetDebianVersion(t *testing.T) {
	t.Parallel()

	v, err := GetDebianVersion("bash")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(v)

	_, err = GetDebianVersion("no-such-package")
	if err == nil {
		t.Error("GetDebianVersion succeeded for non-existing package")
	}
}
