package necorebooter

import (
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	fileContent := `
rebootTimes:
  - name: test1
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test1
    times:
      deny:
        - "* 0-23 1 * *"
      allow:
        - "* 0-7 * * 1-5"
        - "* 19-23 * * 1-5"
  - name: test2
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test2
    times:
      allow:
        - "* 0-23 * * 1-5"
groupLabelKey: topology.kubernetes.io/zone
metricsPort: 9102
`
	config, err := LoadConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	if len(config.RebootTimes) != 2 {
		t.Error("len(config.RebootTimes) != 2, actual ", len(config.RebootTimes))
	}
	if config.GroupLabelKey != "topology.kubernetes.io/zone" {
		t.Error("GroupLabelKey is not expected value")
	}
	if config.MetricsPort != 9102 {
		t.Error("MetricsPort is not expected value")
	}
	for _, rt := range config.RebootTimes {
		if rt.Name == "test1" {
			if rt.LabelSelector.MatchLabels["cke.cybozu.com/role"] != "test1" {
				t.Error("LabelSelector is not expected value")
			}
			if len(rt.Times.Deny) != 1 {
				t.Error("len(rt.Times.Deny) != 1, actual ", len(rt.Times.Deny))
			}
			if len(rt.Times.Allow) != 2 {
				t.Error("len(rt.Times.Allow) != 2, actual ", len(rt.Times.Allow))
			}
		}
		if rt.Name == "test2" {
			if rt.LabelSelector.MatchLabels["cke.cybozu.com/role"] != "test2" {
				t.Error("LabelSelector is not expected value")
			}
			if len(rt.Times.Deny) != 0 {
				t.Error("len(rt.Times.Deny) != 0, actual ", len(rt.Times.Deny))
			}
			if len(rt.Times.Allow) != 1 {
				t.Error("len(rt.Times.Allow) != 1, actual ", len(rt.Times.Allow))
			}
		}
	}
}

func TestGetRebootTime(t *testing.T) {
	fileContent := `
rebootTimes:
  - name: test1
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test1
    times:
      deny:
        - "* 0-23 1 * *"
      allow:
        - "* 0-7 * * 1-5"
        - "* 19-23 * * 1-5"
  - name: test2
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test2
    times:
      allow:
        - "* 0-23 * * 1-5"
groupLabelKey: topology.kubernetes.io/zone
`
	config, err := LoadConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	if len(config.RebootTimes) != 2 {
		t.Error("len(config.RebootTimes) != 2, actual ", len(config.RebootTimes))
	}
	rt, err := config.GetRebootTime()
	if err != nil {
		t.Fatal(err)
	}
	if len(rt) != 2 {
		t.Error("len(rt) != 2, actual ", len(rt))
	}
	if len((rt)["test1"].Deny) != 1 {
		t.Error("len(rt[test1].Deny) != 1, actual ", len((rt)["test1"].Deny))
	}
	if len((rt)["test1"].Allow) != 2 {
		t.Error("len(rt[test1].Allow) != 2, actual ", len((rt)["test1"].Allow))
	}
	if len((rt)["test2"].Deny) != 0 {
		t.Error("len(rt[test2].Deny) != 0, actual ", len((rt)["test2"].Deny))
	}
	if len((rt)["test2"].Allow) != 1 {
		t.Error("len(rt[test2].Allow) != 1, actual ", len((rt)["test2"].Allow))
	}
	fileContent2 := `
rebootTimes:
  - name: test1
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test1
    times:
      deny:
        - "* hoge 1a * *"
groupLabelKey: topology.kubernetes.io/zone
`
	config, err = LoadConfig(strings.NewReader(fileContent2))
	if err != nil {
		t.Fatal(err)
	}
	rt, err = config.GetRebootTime()
	if err == nil {
		t.Error("it should be raised an error")
	}
	if rt != nil {
		t.Error("rt should be nil")
	}
}
