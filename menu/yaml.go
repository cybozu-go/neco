package menu

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cybozu-go/netutil"
	"github.com/cybozu-go/placemat"
	"github.com/cybozu-go/sabakan/v2"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

type baseConfig struct {
	Kind string `json:"kind"`
}

type networkConfig struct {
	Spec struct {
		IPAMConfig    string   `json:"ipam-config"`
		ASNBase       int      `json:"asn-base"`
		Internet      string   `json:"internet"`
		SpineTor      string   `json:"spine-tor"`
		CoreSpine     string   `json:"core-spine"`
		CoreExternal  string   `json:"core-external"`
		CoreOperation string   `json:"core-operation"`
		Proxy         string   `json:"proxy"`
		NTP           []string `json:"ntp"`
		Pod           string   `json:"pod"`
		Exposed       struct {
			Bastion      string `json:"bastion"`
			LoadBalancer string `json:"loadbalancer"`
			Ingress      string `json:"ingress"`
			Global       string `json:"global"`
		} `json:"exposed"`
	} `json:"spec"`
}

type inventoryConfig struct {
	Spec struct {
		ClusterID string `json:"cluster-id"`
		Spine     int    `json:"spine"`
		Rack      []struct {
			CS int `json:"cs"`
			SS int `json:"ss"`
		} `json:"rack"`
	} `json:"spec"`
}

type imageSpec = placemat.ImageSpec

type nodeConfig struct {
	Type string `json:"type"`
	Spec struct {
		CPU               int      `json:"cpu"`
		Memory            string   `json:"memory"`
		Image             string   `json:"image"`
		Data              []string `json:"data"`
		UEFI              bool     `json:"uefi"`
		CloudInitTemplate string   `json:"cloud-init-template"`
		TPM               bool     `json:"tpm"`
	} `json:"spec"`
}

var nodeType = map[string]NodeType{
	"boot": BootNode,
	"cs":   CSNode,
	"ss":   SSNode,
}

func parseNetworkCIDR(s string) (net.IP, *net.IPNet, error) {
	ip, network, err := net.ParseCIDR(s)
	if err != nil {
		return nil, nil, err
	}
	if !ip.Equal(network.IP) {
		return nil, nil, errors.New("Host part of network address must be 0: " + s)
	}
	return ip, network, nil
}

func unmarshalNetwork(dir string, data []byte) (*NetworkMenu, error) {
	var n networkConfig
	err := yaml.Unmarshal(data, &n)
	if err != nil {
		return nil, err
	}

	var network NetworkMenu

	network.IPAMConfigFile = n.Spec.IPAMConfig
	name := network.IPAMConfigFile
	if !filepath.IsAbs(name) {
		name = filepath.Join(dir, name)
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var ic sabakan.IPAMConfig
	err = json.NewDecoder(f).Decode(&ic)
	if err != nil {
		return nil, err
	}
	if ic.NodeIPPerNode != torPerRack+1 {
		return nil, fmt.Errorf("node-ip-per-node in IPAM config must be %d", torPerRack+1)
	}
	if ic.NodeIndexOffset != offsetNodenetBoot {
		return nil, fmt.Errorf("node-index-offset in IPAM config must be %d", offsetNodenetBoot)
	}
	nodePool, _, err := parseNetworkCIDR(ic.NodeIPv4Pool)
	if err != nil {
		return nil, err
	}
	var nodeOffset int64
	if len(ic.NodeIPv4Offset) > 0 {
		bl := strings.Split(ic.NodeIPv4Offset, ".")
		for _, b := range bl {
			nodeOffset <<= 8
			n, err := strconv.ParseInt(b, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid node-ipv4-offset: %s", ic.NodeIPv4Offset)
			}
			nodeOffset += n
		}
	}
	network.NodeBase = netutil.IPAdd(nodePool, nodeOffset)
	network.NodeRangeSize = int(ic.NodeRangeSize)
	network.NodeRangeMask = int(ic.NodeRangeMask)
	_, network.BMC, err = parseNetworkCIDR(ic.BMCIPv4Pool)
	if err != nil {
		return nil, err
	}

	network.ASNBase = n.Spec.ASNBase

	_, network.Internet, err = parseNetworkCIDR(n.Spec.Internet)
	if err != nil {
		return nil, err
	}

	_, network.CoreOperation, err = parseNetworkCIDR(n.Spec.CoreOperation)
	if err != nil {
		return nil, err
	}
	_, network.CoreSpine, err = parseNetworkCIDR(n.Spec.CoreSpine)
	if err != nil {
		return nil, err
	}
	_, network.CoreExternal, err = parseNetworkCIDR(n.Spec.CoreExternal)
	if err != nil {
		return nil, err
	}
	network.SpineTor = net.ParseIP(n.Spec.SpineTor)
	if network.SpineTor == nil {
		return nil, errors.New("Invalid IP address of Spine ToR: " + n.Spec.SpineTor)
	}

	// `proxy` is optional value
	if len(n.Spec.Proxy) != 0 {
		network.Proxy = net.ParseIP(n.Spec.Proxy)
		if network.Proxy == nil {
			return nil, errors.New("Invalid IP address of proxy: " + n.Spec.Proxy)
		}
	}
	// `ntp` is optional value
	if len(n.Spec.NTP) != 0 {
		for _, address := range n.Spec.NTP {
			ntp := net.ParseIP(address)
			if ntp == nil {
				return nil, errors.New("Invalid IP address of ntp: " + address)
			}
			network.NTP = append(network.NTP, ntp)
		}
	}

	_, network.Pod, err = parseNetworkCIDR(n.Spec.Pod)
	if err != nil {
		return nil, err
	}
	_, network.Bastion, err = parseNetworkCIDR(n.Spec.Exposed.Bastion)
	if err != nil {
		return nil, err
	}
	_, network.LoadBalancer, err = parseNetworkCIDR(n.Spec.Exposed.LoadBalancer)
	if err != nil {
		return nil, err
	}
	_, network.Ingress, err = parseNetworkCIDR(n.Spec.Exposed.Ingress)
	if err != nil {
		return nil, err
	}
	_, network.Global, err = parseNetworkCIDR(n.Spec.Exposed.Global)
	if err != nil {
		return nil, err
	}

	return &network, nil
}

func unmarshalInventory(data []byte) (*InventoryMenu, error) {
	var i inventoryConfig
	err := yaml.Unmarshal(data, &i)
	if err != nil {
		return nil, err
	}

	var inventory InventoryMenu

	if i.Spec.ClusterID == "" {
		return nil, errors.New("cluster-id is empty")
	}
	inventory.ClusterID = i.Spec.ClusterID

	if !(i.Spec.Spine > 0) {
		return nil, errors.New("spine in Inventory must be more than 0")
	}
	inventory.Spine = i.Spec.Spine

	inventory.Rack = []RackMenu{}
	for _, r := range i.Spec.Rack {
		var rack RackMenu
		rack.CS = r.CS
		rack.SS = r.SS
		inventory.Rack = append(inventory.Rack, rack)
	}

	return &inventory, nil
}

func unmarshalImage(data []byte) (*imageSpec, error) {
	var i imageSpec
	err := yaml.Unmarshal(data, &i)
	if err != nil {
		return nil, err
	}

	return &i, nil
}

func unmarshalNode(data []byte) (*NodeMenu, error) {
	var n nodeConfig
	err := yaml.Unmarshal(data, &n)
	if err != nil {
		return nil, err
	}

	var node NodeMenu

	nodetype, ok := nodeType[n.Type]
	if !ok {
		return nil, errors.New("Unknown node type: " + n.Type)
	}
	node.Type = nodetype

	if !(n.Spec.CPU > 0) {
		return nil, errors.New("cpu in Node must be more than 0")
	}
	node.CPU = n.Spec.CPU

	node.Memory = n.Spec.Memory
	node.Image = n.Spec.Image
	node.Data = n.Spec.Data
	node.UEFI = n.Spec.UEFI
	node.CloudInitTemplate = n.Spec.CloudInitTemplate
	node.TPM = n.Spec.TPM

	return &node, nil
}

// ReadYAML read placemat-menu resource files
func ReadYAML(dir string, r *bufio.Reader) (*Menu, error) {
	var m Menu
	var c baseConfig
	y := k8sYaml.NewYAMLReader(r)
	for {
		data, err := y.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(data, &c)
		if err != nil {
			return nil, err
		}

		switch c.Kind {
		case "Network":
			r, err := unmarshalNetwork(dir, data)
			if err != nil {
				return nil, err
			}
			m.Network = r
		case "Inventory":
			r, err := unmarshalInventory(data)
			if err != nil {
				return nil, err
			}
			m.Inventory = r
		case "Image":
			r, err := unmarshalImage(data)
			if err != nil {
				return nil, err
			}
			m.Images = append(m.Images, r)
		case "Node":
			r, err := unmarshalNode(data)
			if err != nil {
				return nil, err
			}
			m.Nodes = append(m.Nodes, r)
		default:
			return nil, errors.New("unknown resource: " + c.Kind)
		}
	}
	return &m, nil
}
