package updater

import (
	"context"
	"testing"
)

func TestInstalledNecoVersion(t *testing.T) {
	t.Skip()

	name := "linux-base"
	mgr := DebPackageManager{}
	ver, err := mgr.GetVersion(context.Background(), name)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s's version is `%s'", name, ver)
}
