package sabakan

import (
	"bytes"
	"testing"

	"github.com/cybozu-go/neco"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"
)

func TestGenerateConf(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	err := GenerateConf(buf, 0, []int{0, 1, 2})
	if err != nil {
		t.Fatal(err)
	}

	var actual map[string]interface{}
	err = yaml.Unmarshal(buf.Bytes(), &actual)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]interface{}{
		"advertise-url": "http://" + neco.BootNode0IP(0).String() + ":10080",
		"dhcp-bind":     "0.0.0.0:67",
		"etcd": map[string]interface{}{
			"endpoints": []interface{}{
				"https://" + neco.BootNode0IP(0).String() + ":2379",
				"https://" + neco.BootNode0IP(1).String() + ":2379",
				"https://" + neco.BootNode0IP(2).String() + ":2379",
			},
			"tls-cert-file": neco.SabakanCertFile,
			"tls-key-file":  neco.SabakanKeyFile,
		},
	}

	if !cmp.Equal(actual, expected) {
		t.Error(`unexpected config file`, cmp.Diff(actual, expected))
	}

}
