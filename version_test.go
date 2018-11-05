package neco

import (
	"context"
	"testing"
)

func TestInstalledNecoVersion(t *testing.T) {
	t.Skip()

	packageName = "linux-base"

	ver, err := InstalledNecoVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Version = `%s'", ver)
}
