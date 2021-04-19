package promtail

import (
	"bytes"
	"testing"

	"github.com/cybozu-go/neco"
	"sigs.k8s.io/yaml"
)

func TestGenerateConf(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	err := GenerateConf(buf, 0)
	if err != nil {
		t.Fatal(err)
	}

	var conf struct {
		ScrapeConfigs []struct {
			Journal struct {
				Labels struct {
					Instance string `json:"instance"`
				} `json:"labels"`
			} `json:"journal"`
		} `json:"scrape_configs"`
	}

	err = yaml.Unmarshal(buf.Bytes(), &conf)
	if err != nil {
		t.Fatal(err)
	}

	actual := conf.ScrapeConfigs[0].Journal.Labels.Instance
	if actual != neco.BootNode0IP(0).String() {
		t.Error("unexpected ServerIP", actual)
	}
}
