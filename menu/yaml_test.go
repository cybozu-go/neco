package menu

import (
	"net"
	"reflect"
	"testing"
)

func mustParseCIDR(s string) *net.IPNet {
	_, net, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return net
}

func testUnmarshalNetwork(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source   string
		expected NetworkMenu
	}{
		{
			source: `
kind: Network
spec:
  ipam-config: example/ipam.json
  asn-base: 64600
  internet: 10.0.0.0/24
  core-spine: 10.0.2.0/24
  core-external: 10.0.3.0/24
  core-operation: 10.0.4.0/24
  spine-tor: 10.0.1.0
  proxy: 10.0.49.3
  pod: 10.64.0.0/14
  exposed:
    loadbalancer: 10.72.32.0/20
    bastion: 10.72.48.0/26
    ingress: 10.72.48.64/26
    global: 172.19.0.0/24
`,
			expected: NetworkMenu{
				IPAMConfigFile: "example/ipam.json",
				NodeBase:       net.ParseIP("10.69.0.0").To4(),
				NodeRangeSize:  6,
				NodeRangeMask:  26,
				BMC:            mustParseCIDR("10.72.16.0/20"),
				ASNBase:        64600,
				Internet:       mustParseCIDR("10.0.0.0/24"),
				CoreSpine:      mustParseCIDR("10.0.2.0/24"),
				CoreExternal:   mustParseCIDR("10.0.3.0/24"),
				CoreOperation:  mustParseCIDR("10.0.4.0/24"),
				SpineTor:       net.ParseIP("10.0.1.0"),
				Proxy:          net.ParseIP("10.0.49.3"),
				Pod:            mustParseCIDR("10.64.0.0/14"),
				Bastion:        mustParseCIDR("10.72.48.0/26"),
				LoadBalancer:   mustParseCIDR("10.72.32.0/20"),
				Ingress:        mustParseCIDR("10.72.48.64/26"),
				Global:         mustParseCIDR("172.19.0.0/24"),
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalNetwork(".", []byte(c.source))
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorSources := []string{
		`
# Invalid CIDR @ internet
kind: Network
spec:
  ipam-config: example/ipam.json
  asn-base: 64600
  internet: 10.0.0.0
  spine-tor: 10.0.1.0
  exposed:
    loadbalancer: 10.72.32.0/20
    bastion: 10.72.48.0/26
    ingress: 10.72.48.64/26
    global: 172.19.0.0/24
`,
		`
# Invalid IP address @ spine-tor
kind: Network
spec:
  ipam-config: example/ipam.json
  asn-base: 64600
  internet: 10.0.0.0/24
  spine-tor: 10.0.1.0/31
  exposed:
    loadbalancer: 10.72.32.0/20
    bastion: 10.72.48.0/26
    ingress: 10.72.48.64/26
`,
	}

	for _, s := range errorSources {
		_, err := unmarshalNetwork(".", []byte(s))
		if err == nil {
			t.Error("err == nil", s)
		}
	}
}

func testUnmarshalInventory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source   string
		expected InventoryMenu
	}{
		{
			source: `
kind: Inventory
spec:
  cluster-id: dev0
  spine: 3
  rack:
    - cs: 3
      ss: 0
    - cs: 2
      ss: 2
    - cs: 0
      ss: 3
`,
			expected: InventoryMenu{
				ClusterID: "dev0",
				Spine:     3,
				Rack: []RackMenu{
					{CS: 3, SS: 0},
					{CS: 2, SS: 2},
					{CS: 0, SS: 3},
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalInventory([]byte(c.source))
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorSources := []string{
		`
# No spine
kind: Inventory
spec:
  spine: 0
  rack:
    - cs: 3
      ss: 0
`,
	}

	for _, s := range errorSources {
		_, err := unmarshalInventory([]byte(s))
		if err == nil {
			t.Error("err == nil", s)
		}
	}
}

func testUnmarshalNode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source   string
		expected NodeMenu
	}{
		{
			source: `
kind: Node
type: boot
spec:
  cpu: 1
  memory: 2G
`,
			expected: NodeMenu{
				Type:   BootNode,
				CPU:    1,
				Memory: "2G",
			},
		},
		{
			source: `
kind: Node
type: cs
spec:
  cpu: 2
  memory: 4G
`,
			expected: NodeMenu{
				Type:   CSNode,
				CPU:    2,
				Memory: "4G",
			},
		},
		{
			source: `
kind: Node
type: ss
spec:
  cpu: 1
  memory: 1G
  tpm: true
`,
			expected: NodeMenu{
				Type:   SSNode,
				CPU:    1,
				Memory: "1G",
				TPM:    true,
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalNode([]byte(c.source))
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorSources := []string{
		`
# Invalid type
kind: Node
type: storage
spec:
  cpu: 2
  memory: 2G
`,
		`
# No CPU
kind: Node
type: cs
spec:
  cpu: 0
  memory: 2G
`,
	}

	for _, s := range errorSources {
		_, err := unmarshalNode([]byte(s))
		if err == nil {
			t.Error("err == nil", s)
		}
	}
}

func TestYAML(t *testing.T) {
	t.Run("network", testUnmarshalNetwork)
	t.Run("inventory", testUnmarshalInventory)
	t.Run("node", testUnmarshalNode)
}
