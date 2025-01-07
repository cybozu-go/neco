package menu

import (
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
	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/sabakan/v3"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/yaml"
)

type networkMenu struct {
	Kind string      `json:"kind"`
	Spec networkSpec `json:"spec"`
}

func (n *networkMenu) newNetworkMenu(menuFileDir string) (*network, error) {
	network := &network{
		ipamConfigFile: n.Spec.IpamConfig,
	}

	ipamConfigFile := network.ipamConfigFile
	if !filepath.IsAbs(ipamConfigFile) {
		ipamConfigFile = filepath.Join(menuFileDir, ipamConfigFile)
	}
	f, err := os.Open(ipamConfigFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var ic sabakan.IPAMConfig
	err = json.NewDecoder(f).Decode(&ic)
	if err != nil {
		return nil, err
	}
	if ic.NodeIPPerNode != 3 {
		return nil, fmt.Errorf("node-ip-per-node in IPAM config must be %d", 3)
	}
	if ic.NodeIndexOffset != 3 {
		return nil, fmt.Errorf("node-index-offset in IPAM config must be %d", 3)
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
	network.nodeBase = netutil.IPAdd(nodePool, nodeOffset)
	network.nodeRangeSize = int(ic.NodeRangeSize)
	network.nodeRangeMask = int(ic.NodeRangeMask)
	_, network.bmc, err = parseNetworkCIDR(ic.BMCIPv4Pool)
	if err != nil {
		return nil, err
	}

	network.asnBase = n.Spec.AsnBase
	network.asnCore = n.Spec.AsnBase + offsetASNCore
	network.asnExternal = n.Spec.AsnBase + offsetASNExternal
	network.asnSpine = n.Spec.AsnBase + offsetASNSpine

	_, network.internet, err = parseNetworkCIDR(n.Spec.Internet)
	if err != nil {
		return nil, err
	}

	_, network.coreOperation, err = parseNetworkCIDR(n.Spec.CoreOperation)
	if err != nil {
		return nil, err
	}
	_, network.coreSpine, err = parseNetworkCIDR(n.Spec.CoreSpine)
	if err != nil {
		return nil, err
	}
	_, network.coreExternal, err = parseNetworkCIDR(n.Spec.CoreExternal)
	if err != nil {
		return nil, err
	}
	network.spineTor = net.ParseIP(n.Spec.SpineTor)
	if network.spineTor == nil {
		return nil, errors.New("Invalid IP address of Spine ToR: " + n.Spec.SpineTor)
	}

	if len(n.Spec.Proxy) != 0 {
		network.proxy = net.ParseIP(n.Spec.Proxy)
		if network.proxy == nil {
			return nil, errors.New("Invalid IP address of proxy: " + n.Spec.Proxy)
		}
	}

	if len(n.Spec.Ntp) != 0 {
		_, network.ntp, err = parseNetworkCIDR(n.Spec.Ntp)
		if err != nil {
			return nil, err
		}
	}

	_, network.pod, err = parseNetworkCIDR(n.Spec.Pod)
	if err != nil {
		return nil, err
	}
	_, network.bastion, err = parseNetworkCIDR(n.Spec.Exposed.Bastion)
	if err != nil {
		return nil, err
	}
	_, network.loadBalancer, err = parseNetworkCIDR(n.Spec.Exposed.Loadbalancer)
	if err != nil {
		return nil, err
	}
	_, network.egress, err = parseNetworkCIDR(n.Spec.Exposed.Egress)
	if err != nil {
		return nil, err
	}
	_, network.global, err = parseNetworkCIDR(n.Spec.Exposed.Global)
	if err != nil {
		return nil, err
	}

	return network, nil
}

type networkSpec struct {
	IpamConfig    string  `json:"ipam-config"`
	AsnBase       int     `json:"asn-base"`
	Internet      string  `json:"internet"`
	SpineTor      string  `json:"spine-tor"`
	CoreSpine     string  `json:"core-spine"`
	CoreExternal  string  `json:"core-external"`
	CoreOperation string  `json:"core-operation"`
	Proxy         string  `json:"proxy"`
	Ntp           string  `json:"ntp"`
	Pod           string  `json:"pod"`
	Exposed       exposed `json:"exposed"`
}

type exposed struct {
	Loadbalancer string `json:"loadbalancer"`
	Bastion      string `json:"bastion"`
	Egress       string `json:"egress"`
	Global       string `json:"global"`
}

type inventory struct {
	Kind string        `json:"kind"`
	Spec inventorySpec `json:"spec"`
}

func (i *inventory) validate() error {
	if !(i.Spec.Spine > 0) {
		return errors.New("spine in Inventory must be more than 0")
	}

	return nil
}

type inventorySpec struct {
	ClusterID string      `json:"cluster-id"`
	Spine     int         `json:"spine"`
	Rack      []*rackMenu `json:"rack"`
}

type rackMenu struct {
	Cs  int `json:"cs"`
	Ss  int `json:"ss"`
	Ss2 int `json:"ss2"`
}

type nodeMenu struct {
	Kind string   `json:"kind"`
	Type nodeType `json:"type"`
	Spec nodeSpec `json:"spec"`
}

func (n *nodeMenu) validate() error {
	if n.Spec.CPU == 0 && n.Spec.SMP == nil {
		return errors.New("cpu in Node must be more than 0")
	}
	if err := n.Spec.validate(); err != nil {
		return err
	}

	return nil
}

type smpSpec struct {
	CPUs    int `json:"cpus"`
	Cores   int `json:"cores"`
	Threads int `json:"threads"`
	Dies    int `json:"dies"`
	Sockets int `json:"sockets"`
}

type numaSpec struct {
	Nodes int `json:"nodes"`
}

type diskSpec struct {
	DeviceClass string `json:"device-class"`
	Count       int    `json:"count"`
	Size        string `json:"size"`
}

type nodeSpec struct {
	CPU               int        `json:"cpu"`
	SMP               *smpSpec   `json:"smp"`
	Memory            string     `json:"memory"`
	NUMA              numaSpec   `json:"numa"`
	DiskCount         int        `json:"disk-count"`
	DiskSize          string     `json:"disk-size"`
	Disks             []diskSpec `json:"disks"`
	Image             string     `json:"image"`
	Data              []string   `json:"data"`
	UEFI              bool       `json:"uefi"`
	CloudInitTemplate string     `json:"cloud-init-template"`
	TPM               bool       `json:"tpm"`
}

func (n *nodeSpec) validate() error {
	if n.DiskCount > 0 && len(n.Disks) > 0 {
		return errors.New("DiskCount and Disks are exclusive")
	}

	return nil
}

type nodeType string

const (
	nodeTypeBoot = nodeType("boot")
	nodeTypeCS   = nodeType("cs")
	nodeTypeSS   = nodeType("ss")
	nodeTypeSS2  = nodeType("ss2")
)

type menu struct {
	network       *network
	inventory     *inventory
	images        []*types.ImageSpec
	deviceclasses []*types.DeviceClassSpec
	nodes         []*nodeMenu
}

type baseConfig struct {
	Kind string `json:"kind"`
}

// Parse reads a yaml document and create menu
func Parse(r io.Reader, menuFileDir string) (*menu, error) {
	menu := &menu{}
	f := k8sjson.YAMLFramer.NewFrameReader(io.NopCloser(r))
	for {
		y, err := readSingleYamlDoc(f)
		if err == io.EOF {
			break
		}
		b := &baseConfig{}
		if err := yaml.Unmarshal(y, b); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the yaml document %s: %w", y, err)
		}

		switch b.Kind {
		case "Network":
			n := &networkMenu{}
			if err := yaml.Unmarshal(y, n); err != nil {
				return nil, fmt.Errorf("failed to unmarshal the Network yaml document %s: %w", y, err)
			}
			network, err := n.newNetworkMenu(menuFileDir)
			if err != nil {
				return nil, err
			}
			menu.network = network
		case "Inventory":
			i := &inventory{}
			if err := yaml.Unmarshal(y, i); err != nil {
				return nil, fmt.Errorf("failed to unmarshal the Inventory yaml document %s: %w", y, err)
			}
			if err := i.validate(); err != nil {
				return nil, err
			}
			menu.inventory = i
		case "Node":
			n := &nodeMenu{}
			if err := yaml.Unmarshal(y, n); err != nil {
				return nil, fmt.Errorf("failed to unmarshal the Node yaml document %s: %w", y, err)
			}
			if err := n.validate(); err != nil {
				return nil, err
			}
			menu.nodes = append(menu.nodes, n)
		case "Image":
			i := &types.ImageSpec{}
			if err := yaml.Unmarshal(y, i); err != nil {
				return nil, fmt.Errorf("failed to unmarshal the Image yaml document %s: %w", y, err)
			}
			menu.images = append(menu.images, i)
		case "DeviceClass":
			d := &types.DeviceClassSpec{}
			if err := yaml.Unmarshal(y, d); err != nil {
				return nil, fmt.Errorf("failed to unmarshal the DeviceClass yaml document %s: %w", y, err)
			}
			menu.deviceclasses = append(menu.deviceclasses, d)
		default:
			return nil, errors.New("unknown resource: " + b.Kind)
		}
	}
	return menu, nil
}

func readSingleYamlDoc(reader io.Reader) ([]byte, error) {
	buf := make([]byte, 1024)
	maxBytes := 16 * 1024 * 1024
	base := 0
	for {
		n, err := reader.Read(buf[base:])
		if err == io.ErrShortBuffer {
			if n == 0 {
				return nil, fmt.Errorf("got short buffer with n=0, base=%d, cap=%d", base, cap(buf))
			}
			if len(buf) < maxBytes {
				base += n
				buf = append(buf, make([]byte, len(buf))...)
				continue
			}
			return nil, errors.New("yaml document is too large")
		}
		if err != nil {
			return nil, err
		}
		base += n
		return buf[:base], nil
	}
}
