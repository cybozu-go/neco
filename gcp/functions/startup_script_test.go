package functions

import (
	"strings"
	"testing"
)

func TestNecoStartupScriptBuilder(t *testing.T) {
	_, err := NewNecoStartupScriptBuilder().WithNecoApps("this-is-neco-apps")
	if err == nil {
		t.Errorf("should fail when neco-apps is enabled without neco")
	}

	builder, err := NewNecoStartupScriptBuilder().
		WithFluentd().
		WithNeco("this-is-neco").
		WithNecoApps("this-is-neco-apps")
	if err != nil {
		t.Errorf("should not fail because neco-apps is enabled after neco is enabled")
	}

	s := builder.Build()
	shouldContain := []string{
		"mkfs -t ext4 -F /dev/disk/by-id/google-local-ssd-0",
		"service google-fluentd restart", // check for .WithFluentd
		"git checkout this-is-neco",      // check for .WithNeco
		"git checkout this-is-neco-apps", // check for .WithNecoApps
	}
	for _, v := range shouldContain {
		if !strings.Contains(s, v) {
			t.Errorf("should contain %q", v)
		}
	}
}
