package menu

import (
	"fmt"
	"io"

	"github.com/cybozu-go/placemat"
	"sigs.k8s.io/yaml"
)

const (
	dockerImageBird    = "docker://quay.io/cybozu/bird:2.0"
	dockerImageDebug   = "docker://quay.io/cybozu/ubuntu-debug:18.04"
	dockerImageDnsmasq = "docker://quay.io/cybozu/dnsmasq:2.79"
	dockerImageSquid   = "docker://quay.io/cybozu/squid:3.5"
	dockerImageChrony  = "docker://quay.io/cybozu/chrony:3.5"
)

var birdContainer = placemat.PodAppSpec{
	Name:           "bird",
	Image:          dockerImageBird,
	ReadOnlyRootfs: true,
	Mount: []placemat.PodAppMountSpec{
		{
			Volume: "config",
			Target: "/etc/bird",
		},
		{
			Volume: "run",
			Target: "/run/bird",
		},
	},
	CapsRetain: []string{
		"CAP_NET_ADMIN",
		"CAP_NET_BIND_SERVICE",
		"CAP_NET_RAW",
	},
}

var debugContainer = placemat.PodAppSpec{
	Name:           "debug",
	Image:          dockerImageDebug,
	ReadOnlyRootfs: true,
}

type cluster struct {
	networks    []*placemat.NetworkSpec
	dataFolders []*placemat.DataFolderSpec
	pods        []*placemat.PodSpec
	nodes       []*placemat.NodeSpec
}

// ExportCluster exports a placemat configuration to writer from TemplateArgs
func ExportCluster(w io.Writer, ta *TemplateArgs) error {
	cluster := generateCluster(ta)

	isFirstDocument := true
	writeDocument := func(o interface{}) error {
		if !isFirstDocument {
			_, err := w.Write([]byte("---\n"))
			if err != nil {
				return err
			}
		}
		isFirstDocument = false
		data, err := yaml.Marshal(o)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		if err != nil {
			return err
		}
		return nil
	}

	for _, n := range cluster.networks {
		err := writeDocument(n)
		if err != nil {
			return err
		}
	}
	for _, i := range ta.Images {
		err := writeDocument(i)
		if err != nil {
			return err
		}
	}
	for _, f := range cluster.dataFolders {
		err := writeDocument(f)
		if err != nil {
			return err
		}
	}
	for _, n := range cluster.nodes {
		err := writeDocument(n)
		if err != nil {
			return err
		}
	}
	for _, p := range cluster.pods {
		err := writeDocument(p)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateCluster(ta *TemplateArgs) *cluster {
	cluster := new(cluster)

	cluster.appendExternalNetwork(ta)

	cluster.appendCoreNetwork(ta)

	cluster.appendBMCNetwork(ta)

	cluster.appendSpineToRackNetwork(ta)

	cluster.appendRackNetwork(ta)

	cluster.appendCoreDataFolder(ta)

	cluster.appendSpineDataFolder(ta)

	cluster.appendRackDataFolder(ta)

	cluster.appendSabakanDataFolder()

	cluster.appendCorePod(ta)

	cluster.appendSpinePod(ta)

	cluster.appendToRPods(ta)

	cluster.appendExtPod(ta)

	cluster.appendOperationPod(ta)

	cluster.appendNodes(ta)

	return cluster
}

func (c *cluster) appendOperationPod(ta *TemplateArgs) {
	pod := &placemat.PodSpec{
		Kind:        "Pod",
		Name:        "operation",
		InitScripts: []string{"setup-default-gateway-operation"},
		Interfaces: []placemat.PodInterfaceSpec{
			{
				Network:   "core-to-op",
				Addresses: []string{ta.Network.Endpoints.Operation.String()},
			},
		},
		Apps: []*placemat.PodAppSpec{
			{
				Name:  "ubuntu",
				Image: dockerImageDebug,
				Exec:  "/bin/sleep",
				Args:  []string{"infinity"},
			},
		},
	}

	c.pods = append(c.pods, pod)
}

func (c *cluster) appendExtPod(ta *TemplateArgs) {
	pod := &placemat.PodSpec{
		Kind:        "Pod",
		Name:        "external",
		InitScripts: []string{"setup-default-gateway-external"},
		Interfaces: []placemat.PodInterfaceSpec{
			{
				Network:   "core-to-ext",
				Addresses: []string{ta.Network.Endpoints.External.String()},
			},
		},
		Apps: []*placemat.PodAppSpec{
			{
				Name:  "ubuntu",
				Image: dockerImageDebug,
				Exec:  "/bin/sleep",
				Args:  []string{"infinity"},
			},
		},
	}

	c.pods = append(c.pods, pod)
}

func bootNode(rack *Rack, resource *VMResource) *placemat.NodeSpec {
	var volumes []placemat.NodeVolumeSpec
	if resource.Image != "" {
		volumes = []placemat.NodeVolumeSpec{
			{
				Kind:        "image",
				Name:        "root",
				Image:       resource.Image,
				CopyOnWrite: true,
				Cache:       "writeback",
			},
		}
		if resource.CloudInitTemplate != "" {
			volumes = append(volumes, placemat.NodeVolumeSpec{
				Kind:          "localds",
				Name:          "seed",
				UserData:      fmt.Sprintf("seed_%s.yml", rack.BootNode.Fullname),
				NetworkConfig: "network.yml",
			})
		}
	} else {
		volumes = []placemat.NodeVolumeSpec{
			{
				Kind: "raw",
				Name: "root",
				Size: "30G",
			},
		}
	}

	volumes = append(volumes, placemat.NodeVolumeSpec{
		Kind:   "vvfat",
		Name:   "sabakan",
		Folder: "sabakan-data",
	})

	for i, dataImg := range resource.Data {
		volumes = append(volumes, placemat.NodeVolumeSpec{
			Kind:        "image",
			Name:        fmt.Sprintf("extra%d", i),
			Image:       dataImg,
			CopyOnWrite: true,
		})
	}

	return &placemat.NodeSpec{
		Kind: "Node",
		Name: rack.BootNode.Fullname,
		Interfaces: []string{
			fmt.Sprintf("%s-node1", rack.ShortName),
			fmt.Sprintf("%s-node2", rack.ShortName),
		},
		Volumes: volumes,
		CPU:     resource.CPU,
		Memory:  resource.Memory,
		UEFI:    resource.UEFI,
		SMBIOS: placemat.SMBIOSConfig{
			Serial: rack.BootNode.Serial,
		},
		TPM: resource.TPM,
	}
}

func emptyNode(rackName, rackShortName, nodeName, serial string, disks int, resource *VMResource) *placemat.NodeSpec {
	volumes := make([]placemat.NodeVolumeSpec, disks)
	for i := 0; i < disks; i++ {
		volumes[i].Kind = "raw"
		volumes[i].Name = fmt.Sprintf("data%d", i+1)
		volumes[i].Size = "30G"
		volumes[i].Cache = "writeback"
	}

	for i, dataImg := range resource.Data {
		volumes = append(volumes, placemat.NodeVolumeSpec{
			Kind:        "image",
			Name:        fmt.Sprintf("extra%d", i),
			Image:       dataImg,
			CopyOnWrite: true,
		})
	}

	return &placemat.NodeSpec{
		Kind: "Node",
		Name: fmt.Sprintf("%s-%s", rackName, nodeName),
		Interfaces: []string{
			fmt.Sprintf("%s-node1", rackShortName),
			fmt.Sprintf("%s-node2", rackShortName),
		},
		Volumes: volumes,
		CPU:     resource.CPU,
		Memory:  resource.Memory,
		UEFI:    resource.UEFI,
		SMBIOS: placemat.SMBIOSConfig{
			Serial: serial,
		},
		TPM: resource.TPM,
	}
}

func (c *cluster) appendNodes(ta *TemplateArgs) {
	for _, rack := range ta.Racks {
		c.nodes = append(c.nodes, bootNode(&rack, &ta.Boot))

		for _, cp := range rack.CPList {
			c.nodes = append(c.nodes, emptyNode(rack.Name, rack.ShortName, cp.Name, cp.Serial, 2, &ta.CP))
		}
		for _, cs := range rack.CSList {
			c.nodes = append(c.nodes, emptyNode(rack.Name, rack.ShortName, cs.Name, cs.Serial, 2, &ta.CS))
		}
		for _, ss := range rack.SSList {
			c.nodes = append(c.nodes, emptyNode(rack.Name, rack.ShortName, ss.Name, ss.Serial, 4, &ta.SS))
		}
	}
}

func torPod(rackName, rackShortName string, tor ToR, torNumber int, ta *TemplateArgs) *placemat.PodSpec {
	var spineIfs []placemat.PodInterfaceSpec
	for i, spine := range ta.Spines {
		spineIfs = append(spineIfs,
			placemat.PodInterfaceSpec{
				Network:   fmt.Sprintf("%s-to-%s-%d", spine.ShortName, rackShortName, torNumber),
				Addresses: []string{tor.SpineAddresses[i].String()},
			},
		)
	}
	spineIfs = append(spineIfs, placemat.PodInterfaceSpec{
		Network:   fmt.Sprintf("%s-node%d", rackShortName, torNumber),
		Addresses: []string{tor.NodeAddress.String()},
	})

	dhcpRelayArgs := []string{
		"--keep-in-foreground",
		"--pid-file",
		"--log-facility=-",
	}
	for _, r := range ta.Racks {
		if r.Name == rackName {
			continue
		}
		dhcpRelayArgs = append(dhcpRelayArgs, "--dhcp-relay")
		dhcpRelayArgs = append(dhcpRelayArgs, tor.NodeAddress.IP.String()+","+r.BootNode.Node0Address.IP.String())
	}

	return &placemat.PodSpec{
		Kind:       "Pod",
		Name:       fmt.Sprintf("%s-tor%d", rackName, torNumber),
		Interfaces: spineIfs,
		Volumes: []*placemat.PodVolumeSpec{
			{
				Name:     "config",
				Kind:     "host",
				Folder:   fmt.Sprintf("%s-tor%d-data", rackName, torNumber),
				ReadOnly: true,
			},
			{
				Name: "run",
				Kind: "empty",
			},
		},
		Apps: []*placemat.PodAppSpec{
			&birdContainer,
			&debugContainer,
			{
				Name:           "dhcp-relay",
				Image:          dockerImageDnsmasq,
				ReadOnlyRootfs: true,
				CapsRetain: []string{
					"CAP_NET_BIND_SERVICE",
					"CAP_NET_RAW",
					"CAP_NET_BROADCAST",
				},
				Args: dhcpRelayArgs,
			},
		},
	}
}

func (c *cluster) appendToRPods(ta *TemplateArgs) {
	for _, rack := range ta.Racks {
		c.pods = append(c.pods,
			torPod(rack.Name, rack.ShortName, rack.ToR1, 1, ta),
			torPod(rack.Name, rack.ShortName, rack.ToR2, 2, ta),
		)
	}
}

func (c *cluster) appendCorePod(ta *TemplateArgs) {
	var interfaces []placemat.PodInterfaceSpec
	interfaces = append(interfaces, placemat.PodInterfaceSpec{
		Network:   "internet",
		Addresses: []string{ta.Core.InternetAddress.String()},
	})
	interfaces = append(interfaces, placemat.PodInterfaceSpec{
		Network:   "bmc",
		Addresses: []string{ta.Core.BMCAddress.String()},
	})
	if ta.Core.ProxyAddress != nil {
		interfaces = append(interfaces, placemat.PodInterfaceSpec{
			Network:   "core-node0",
			Addresses: []string{ta.Core.ProxyAddress.String() + "/32"},
		})
	}
	if ta.Core.NTPAddresses != nil {
		for i := range ta.Core.NTPAddresses {
			interfaces = append(interfaces, placemat.PodInterfaceSpec{
				Network:   fmt.Sprintf("core-ntp%d", i),
				Addresses: []string{ta.Core.NTPAddresses[i].String() + "/32"},
			})
		}
	}

	for i, spine := range ta.Spines {
		interfaces = append(interfaces, placemat.PodInterfaceSpec{
			Network: fmt.Sprintf("core-to-%s", spine.ShortName),
			Addresses: []string{
				ta.Core.SpineAddresses[i].String(),
			},
		})
	}
	interfaces = append(interfaces, placemat.PodInterfaceSpec{
		Network: "core-to-ext",
		Addresses: []string{
			ta.Core.ExternalAddress.String(),
		},
	})
	interfaces = append(interfaces, placemat.PodInterfaceSpec{
		Network: "core-to-op",
		Addresses: []string{
			ta.Core.OperationAddress.String(),
		},
	})
	coreVolumes := []*placemat.PodVolumeSpec{
		{
			Name:     "config",
			Kind:     "host",
			Folder:   "core-data",
			ReadOnly: true,
		},
		{
			Name: "run",
			Kind: "empty",
		},
	}
	coreApps := []*placemat.PodAppSpec{
		&birdContainer,
		&debugContainer,
	}
	if ta.Core.ProxyAddress != nil {
		coreVolumes = append(coreVolumes, []*placemat.PodVolumeSpec{
			{
				Name: "squid-log",
				Kind: "empty",
				Mode: "0777",
			},
			{
				Name: "squid-spool",
				Kind: "empty",
				Mode: "0777",
			},
		}...)
		coreApps = append(coreApps, &placemat.PodAppSpec{
			Name:  "proxy",
			Image: dockerImageSquid,
			Mount: []placemat.PodAppMountSpec{
				{
					Volume: "config",
					Target: "/etc/squid",
				},
				{
					Volume: "squid-log",
					Target: "/var/log/squid",
				},
				{
					Volume: "squid-spool",
					Target: "/var/spool/squid",
				},
			},
		})
	}
	if ta.Core.NTPAddresses != nil {
		coreVolumes = append(coreVolumes, []*placemat.PodVolumeSpec{
			{
				Name: "chrony-run",
				Kind: "empty",
				Mode: "0777",
			},
			{
				Name: "chrony-var-lib-chrony",
				Kind: "empty",
				Mode: "0777",
			},
		}...)
		coreApps = append(coreApps, &placemat.PodAppSpec{
			Name:  "ntp",
			Image: dockerImageChrony,
			Args:  []string{"-f", "/etc/chrony/chrony.conf"},
			Mount: []placemat.PodAppMountSpec{
				{
					Volume: "config",
					Target: "/etc/chrony",
				},
				{
					Volume: "chrony-run",
					Target: "/run",
				},
				{
					Volume: "chrony-var-lib-chrony",
					Target: "/var/lib/chrony",
				},
			},
		})
	}

	c.pods = append(c.pods, &placemat.PodSpec{
		Kind:        "Pod",
		Name:        "core",
		InitScripts: []string{"setup-iptables"},
		Interfaces:  interfaces,
		Volumes:     coreVolumes,
		Apps:        coreApps,
	})
}

func (c *cluster) appendSpinePod(ta *TemplateArgs) {
	for _, spine := range ta.Spines {
		var ifces []placemat.PodInterfaceSpec

		ifces = append(ifces,
			placemat.PodInterfaceSpec{
				Network:   fmt.Sprintf("core-to-%s", spine.ShortName),
				Addresses: []string{spine.CoreAddress.String()},
			},
		)
		for i, rack := range ta.Racks {
			ifces = append(ifces,
				placemat.PodInterfaceSpec{
					Network:   fmt.Sprintf("%s-to-%s-1", spine.ShortName, rack.ShortName),
					Addresses: []string{spine.ToR1Address(i).String()},
				},
				placemat.PodInterfaceSpec{
					Network:   fmt.Sprintf("%s-to-%s-2", spine.ShortName, rack.ShortName),
					Addresses: []string{spine.ToR2Address(i).String()},
				},
			)
		}

		c.pods = append(c.pods, &placemat.PodSpec{
			Kind:       "Pod",
			Name:       spine.Name,
			Interfaces: ifces,
			Volumes: []*placemat.PodVolumeSpec{
				{
					Name:     "config",
					Kind:     "host",
					Folder:   fmt.Sprintf("%s-data", spine.Name),
					ReadOnly: true,
				},
				{
					Name: "run",
					Kind: "empty",
				},
			},
			Apps: []*placemat.PodAppSpec{
				&birdContainer,
				&debugContainer,
			},
		})
	}
}

func (c *cluster) appendSabakanDataFolder() {
	c.dataFolders = append(c.dataFolders,
		&placemat.DataFolderSpec{
			Kind: "DataFolder",
			Name: "sabakan-data",
			Dir:  "sabakan",
		})
}

func (c *cluster) appendRackDataFolder(ta *TemplateArgs) {
	for _, rack := range ta.Racks {
		c.dataFolders = append(c.dataFolders,
			&placemat.DataFolderSpec{
				Kind: "DataFolder",
				Name: fmt.Sprintf("%s-tor1-data", rack.Name),
				Files: []placemat.DataFolderFileSpec{
					{
						Name: "bird.conf",
						File: fmt.Sprintf("bird_%s-tor1.conf", rack.Name),
					},
				},
			},
			&placemat.DataFolderSpec{
				Kind: "DataFolder",
				Name: fmt.Sprintf("%s-tor2-data", rack.Name),
				Files: []placemat.DataFolderFileSpec{
					{
						Name: "bird.conf",
						File: fmt.Sprintf("bird_%s-tor2.conf", rack.Name),
					},
				},
			},
		)
	}
}

func (c *cluster) appendCoreDataFolder(ta *TemplateArgs) {
	files := []placemat.DataFolderFileSpec{
		{
			Name: "bird.conf",
			File: "bird_core.conf",
		},
	}
	if ta.Core.ProxyAddress != nil {
		files = append(files, placemat.DataFolderFileSpec{
			Name: "squid.conf",
			File: "squid.conf",
		})
	}
	if ta.Core.NTPAddresses != nil {
		files = append(files, placemat.DataFolderFileSpec{
			Name: "chrony.conf",
			File: "chrony.conf",
		})
	}
	c.dataFolders = append(c.dataFolders,
		&placemat.DataFolderSpec{
			Kind:  "DataFolder",
			Name:  "core-data",
			Files: files,
		})
}

func (c *cluster) appendSpineDataFolder(ta *TemplateArgs) {
	for _, spine := range ta.Spines {
		c.dataFolders = append(c.dataFolders,
			&placemat.DataFolderSpec{
				Kind: "DataFolder",
				Name: fmt.Sprintf("%s-data", spine.Name),
				Files: []placemat.DataFolderFileSpec{
					{
						Name: "bird.conf",
						File: fmt.Sprintf("bird_%s.conf", spine.Name),
					},
				},
			})
	}
}

func (c *cluster) appendRackNetwork(ta *TemplateArgs) {
	for _, rack := range ta.Racks {
		c.networks = append(
			c.networks,
			&placemat.NetworkSpec{
				Kind: "Network",
				Name: fmt.Sprintf("%s-node1", rack.ShortName),
				Type: "internal",
			},
			&placemat.NetworkSpec{
				Kind: "Network",
				Name: fmt.Sprintf("%s-node2", rack.ShortName),
				Type: "internal",
			},
		)
	}
}

func (c *cluster) appendSpineToRackNetwork(ta *TemplateArgs) {
	for _, spine := range ta.Spines {
		for _, rack := range ta.Racks {
			c.networks = append(
				c.networks,
				&placemat.NetworkSpec{
					Kind: "Network",
					Name: fmt.Sprintf("%s-to-%s-1", spine.ShortName, rack.ShortName),
					Type: "internal",
				},
				&placemat.NetworkSpec{
					Kind: "Network",
					Name: fmt.Sprintf("%s-to-%s-2", spine.ShortName, rack.ShortName),
					Type: "internal",
				},
			)
		}
	}
}

func (c *cluster) appendExternalNetwork(ta *TemplateArgs) {
	c.networks = append(
		c.networks,
		&placemat.NetworkSpec{
			Kind:    "Network",
			Name:    "internet",
			Type:    "external",
			UseNAT:  true,
			Address: ta.Network.Endpoints.Host.String(),
		},
	)
}

func (c *cluster) appendCoreNetwork(ta *TemplateArgs) {
	for _, spine := range ta.Spines {
		c.networks = append(c.networks, &placemat.NetworkSpec{
			Kind: "Network",
			Name: fmt.Sprintf("core-to-%s", spine.ShortName),
			Type: "internal",
		})
	}

	c.networks = append(
		c.networks,
		&placemat.NetworkSpec{
			Kind: "Network",
			Name: "core-to-ext",
			Type: "internal",
		},
		&placemat.NetworkSpec{
			Kind: "Network",
			Name: "core-to-op",
			Type: "internal",
		},
	)
	if ta.Core.ProxyAddress != nil {
		c.networks = append(
			c.networks,
			&placemat.NetworkSpec{
				Kind: "Network",
				Name: "core-node0",
				Type: "internal",
			},
		)
	}
	if ta.Core.NTPAddresses != nil {
		for i := range ta.Core.NTPAddresses {
			c.networks = append(
				c.networks,
				&placemat.NetworkSpec{
					Kind: "Network",
					Name: fmt.Sprintf("core-ntp%d", i),
					Type: "internal",
				},
			)
		}
	}

}

func (c *cluster) appendBMCNetwork(ta *TemplateArgs) {
	c.networks = append(
		c.networks,
		&placemat.NetworkSpec{
			Kind:    "Network",
			Name:    "bmc",
			Type:    "bmc",
			Address: addToIPNet(ta.Network.BMC, offsetBMCHost).String(),
		},
	)
}

// ExportEmptyNetworkConfig export empty network-config file used in cloud-init
func ExportEmptyNetworkConfig(w io.Writer) error {
	_, err := fmt.Fprintln(w, "version: 2\nethernets: {}")
	return err
}
