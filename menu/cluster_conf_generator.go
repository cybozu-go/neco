package menu

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/cybozu-go/sabakan/v3"
)

const sabakanDir = "sabakan"

type BirdCoreArgs struct {
	Internet string
	ASNCore  int
	ASNSpine int
	Spines   []*SpineArgs
}

type BirdSpineArgs struct {
	SpineIdx         int
	ASNCore          int
	ASNSpine         int
	LoadBalancer     string
	Bastion          string
	Egress           string
	Global           string
	Bmc              string
	BmcAddress       string
	CoreSpineAddress string
	Racks            []*RackArgs
}

type BirdRackArgs struct {
	ASNSpine int
	Spines   []*SpineArgs
	Rack     *RackArgs
}

type MachinesYamlArgs struct {
	Racks []*RackArgs
}

type SeedYamlArgs struct {
	Name string
	Rack *RackArgs
}

type SpineArgs struct {
	Name        string
	CoreAddress string
	ToR1Address string
	ToR2Address string
}

type RackArgs struct {
	Name     string
	Index    int
	ASN      int
	ToR1     ToRArgs
	ToR2     ToRArgs
	BootNode NodeArgs
	CSList   []*NodeArgs
	SSList   []*NodeArgs
	SS2List  []*NodeArgs
}

type NodeArgs struct {
	Name           string
	BastionAddress string
	Node1Address   string
	Node2Address   string
}

type ToRArgs struct {
	NodeInterface  string
	SpineAddresses []string
}

type SetupIPTablesArgs struct {
	Internet string
	NTP      string
}

type ChronyIgnitionArgs struct {
	NTPServers []string
	Gateway    string
	ChronyTag  string
}

func (c *Cluster) generateConfFiles(inputDir, outputDir string, opt *GenerateOption) error {
	if err := c.generateBirdCoreConf(outputDir); err != nil {
		return err
	}

	if err := c.generateBirdRackConf(outputDir); err != nil {
		return err
	}

	if err := c.generateBirdSpineConf(outputDir); err != nil {
		return err
	}

	if err := c.generateMachinesYaml(outputDir); err != nil {
		return err
	}

	if err := c.generateNetworkYaml(outputDir); err != nil {
		return err
	}

	if err := c.generateSeedYaml(inputDir, outputDir); err != nil {
		return err
	}

	if err := c.generateChronyIgnition(outputDir, opt.ChronyTag); err != nil {
		return err
	}

	if err := generateSquidConf(outputDir); err != nil {
		return err
	}

	if err := c.generateSetupScripts(outputDir); err != nil {
		return err
	}

	return c.generateSabakanData(outputDir)
}

func (c *Cluster) generateBirdCoreConf(outputDir string) error {
	args := &BirdCoreArgs{
		Internet: c.internet.address.IP.String(),
		ASNCore:  c.network.asnCore,
		ASNSpine: c.network.asnSpine,
	}

	for _, spine := range c.spines {
		args.Spines = append(args.Spines, &SpineArgs{
			Name:        spine.name,
			CoreAddress: spine.coreAddress.IP.String(),
		})
	}

	return export("/bird_core.conf", filepath.Join(outputDir, "bird_core.conf"), args)
}

func (c *Cluster) generateBirdSpineConf(outputDir string) error {
	for spineIdx := range c.spines {
		args := &BirdSpineArgs{
			SpineIdx:         spineIdx,
			ASNCore:          c.network.asnCore,
			ASNSpine:         c.network.asnSpine,
			LoadBalancer:     c.network.loadBalancer.String(),
			Bastion:          c.network.bastion.String(),
			Egress:           c.network.egress.String(),
			Global:           c.network.global.String(),
			Bmc:              c.network.bmc.String(),
			BmcAddress:       addToIP(c.network.bmc.IP, spineIdx+2, 20).IP.String(),
			CoreSpineAddress: c.core.spineAddresses[spineIdx].IP.String(),
		}

		for _, rack := range c.racks {
			args.Racks = append(args.Racks, newRackArgs(rack))
		}

		if err := export("/bird_spine.conf", filepath.Join(outputDir, fmt.Sprintf("bird_spine%d.conf", spineIdx+1)), args); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) generateBirdRackConf(outputDir string) error {
	for rackIdx, rack := range c.racks {
		args := &BirdRackArgs{
			ASNSpine: c.network.asnSpine,
			Rack:     newRackArgs(rack),
		}

		for _, spine := range c.spines {
			args.Spines = append(args.Spines, &SpineArgs{
				Name:        spine.name,
				CoreAddress: spine.coreAddress.IP.String(),
				ToR1Address: spine.tor1Address(rackIdx).IP.String(),
				ToR2Address: spine.tor2Address(rackIdx).IP.String(),
			})
		}

		if err := export("/bird_rack-tor1.conf", filepath.Join(outputDir, fmt.Sprintf("bird_rack%d-tor1.conf", rackIdx)), args); err != nil {
			return err
		}
		if err := export("/bird_rack-tor2.conf", filepath.Join(outputDir, fmt.Sprintf("bird_rack%d-tor2.conf", rackIdx)), args); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) generateSeedYaml(inputDir, outputDir string) error {
	for rackIdx, rack := range c.racks {
		rackArgs := newRackArgs(rack)
		if rack.bootNode.spec.CloudInitTemplate != "" {
			args := &SeedYamlArgs{
				Name: fmt.Sprintf("boot-%d", rackIdx),
				Rack: rackArgs,
			}
			err := exportFile(filepath.Join(inputDir, rack.bootNode.spec.CloudInitTemplate),
				filepath.Join(outputDir, fmt.Sprintf("seed_boot-%d.yml", rack.index)), args)
			if err != nil {
				return err
			}
		}

		for _, cs := range rack.csList {
			if cs.spec.CloudInitTemplate != "" {
				args := &SeedYamlArgs{
					Name: fmt.Sprintf("%s-%s", rack.name, cs.name),
					Rack: rackArgs,
				}
				err := exportFile(filepath.Join(inputDir, cs.spec.CloudInitTemplate),
					filepath.Join(outputDir, fmt.Sprintf("seed_%s-%s.yml", rack.name, cs.name)), args)
				if err != nil {
					return err
				}
			}
		}

		for _, ss := range rack.ssList {
			if ss.spec.CloudInitTemplate != "" {
				args := &SeedYamlArgs{
					Name: fmt.Sprintf("%s-%s", rack.name, ss.name),
					Rack: rackArgs,
				}
				err := exportFile(filepath.Join(inputDir, ss.spec.CloudInitTemplate),
					filepath.Join(outputDir, fmt.Sprintf("seed_%s-%s.yml", rack.name, ss.name)), args)
				if err != nil {
					return err
				}
			}
		}
		for _, ss2 := range rack.ss2List {
			if ss2.spec.CloudInitTemplate != "" {
				args := &SeedYamlArgs{
					Name: fmt.Sprintf("%s-%s", rack.name, ss2.name),
					Rack: rackArgs,
				}
				err := exportFile(filepath.Join(inputDir, ss2.spec.CloudInitTemplate),
					filepath.Join(outputDir, fmt.Sprintf("seed_%s-%s.yml", rack.name, ss2.name)), args)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (c *Cluster) generateMachinesYaml(outputDir string) error {
	args := &MachinesYamlArgs{}
	for _, rack := range c.racks {
		args.Racks = append(args.Racks, newRackArgs(rack))
	}

	return export("/machines.tpl", filepath.Join(outputDir, "machines.yml"), args)
}

func (c *Cluster) generateNetworkYaml(outputDir string) error {
	networkFile, err := os.Create(filepath.Join(outputDir, "network.yml"))
	if err != nil {
		return err
	}
	defer networkFile.Close()

	_, err = fmt.Fprintln(networkFile, "version: 2\nethernets: {}")
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) generateSetupScripts(outputDir string) error {
	args := &SetupIPTablesArgs{
		Internet: c.network.global.String(),
		NTP:      c.network.ntp.String(),
	}
	if err := exportExecutableFile("/setup-iptables.sh",
		filepath.Join(outputDir, "setup-iptables"), args); err != nil {
		return err
	}

	if err := exportExecutableFile("/setup-iptables-spine.sh", filepath.Join(outputDir, "setup-iptables-spine"), c.bmc.address.String()); err != nil {
		return err
	}

	if err := exportExecutableFile("/setup-default-gateway.sh",
		filepath.Join(outputDir, "setup-default-gateway-operation"), c.core.operationAddress.IP.String()); err != nil {
		return err
	}

	if err := exportExecutableFile("/setup-default-gateway.sh",
		filepath.Join(outputDir, "setup-default-gateway-external"), c.core.externalAddress.IP.String()); err != nil {
		return err
	}

	return nil
}

func (c *Cluster) generateChronyIgnition(outputDir, chronyTag string) error {
	args := &ChronyIgnitionArgs{
		Gateway:   c.core.coreNTPAddress.IP.String(),
		ChronyTag: chronyTag,
	}
	for _, ntpServer := range c.core.ntpServers {
		args.NTPServers = append(args.NTPServers, ntpServer.String())
	}
	if err := export("/chrony-ign.yml",
		filepath.Join(outputDir, "chrony-ign.yml"), args); err != nil {
		return err
	}

	return nil
}

func generateSquidConf(outputDir string) error {
	confFile := "/squid.conf"
	return copyFile(confFile, filepath.Join(outputDir, confFile))
}

func (c *Cluster) generateSabakanData(outputDir string) error {
	var ms []sabakan.MachineSpec
	for _, rack := range c.racks {
		ms = append(ms, sabakanMachine(rack.bootNode.serial, rack.index, "boot"))

		for _, cs := range rack.csList {
			ms = append(ms, sabakanMachine(cs.serial, rack.index, "cs"))
		}
		for _, ss := range rack.ssList {
			ms = append(ms, sabakanMachine(ss.serial, rack.index, "ss"))
		}
		for _, ss2 := range rack.ss2List {
			ms = append(ms, sabakanMachine(ss2.serial, rack.index, "ss2"))
		}
	}

	sabakanDir := filepath.Join(outputDir, sabakanDir)
	err := os.MkdirAll(sabakanDir, 0755)
	if err != nil {
		return err
	}

	return exportJSON(filepath.Join(sabakanDir, "machines.json"), ms)
}

func sabakanMachine(serial string, rack int, role string) sabakan.MachineSpec {
	return sabakan.MachineSpec{
		Serial: serial,
		Labels: map[string]string{
			"product":      "vm",
			"machine-type": "qemu",
			"datacenter":   "dc1",
		},
		Rack: uint(rack),
		Role: role,
		BMC: sabakan.MachineBMC{
			Type: "IPMI-2.0",
		},
	}
}

func exportJSON(dst string, data interface{}) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func newRackArgs(rack *rack) *RackArgs {
	args := &RackArgs{
		Name:  rack.name,
		Index: rack.index,
		ASN:   rack.asn,
		ToR1: ToRArgs{
			NodeInterface: rack.tor1.nodeInterface,
		},
		ToR2: ToRArgs{
			NodeInterface: rack.tor2.nodeInterface,
		},
		BootNode: NodeArgs{
			Name:           rack.bootNode.fullName,
			BastionAddress: rack.bootNode.bastionAddress.String(),
			Node1Address:   rack.bootNode.node1Address.IP.String(),
			Node2Address:   rack.bootNode.node2Address.IP.String(),
		},
	}

	for _, spineAddress := range rack.tor1.spineAddresses {
		args.ToR1.SpineAddresses = append(args.ToR1.SpineAddresses, spineAddress.IP.String())
	}

	for _, spineAddress := range rack.tor2.spineAddresses {
		args.ToR2.SpineAddresses = append(args.ToR2.SpineAddresses, spineAddress.IP.String())
	}

	for _, cs := range rack.csList {
		args.CSList = append(args.CSList, &NodeArgs{
			Name:         cs.name,
			Node1Address: cs.node1Address.IP.String(),
			Node2Address: cs.node2Address.IP.String(),
		})
	}

	for _, ss := range rack.ssList {
		args.SSList = append(args.SSList, &NodeArgs{
			Name:         ss.name,
			Node1Address: ss.node1Address.IP.String(),
			Node2Address: ss.node2Address.IP.String(),
		})
	}
	for _, ss2 := range rack.ss2List {
		args.SS2List = append(args.SS2List, &NodeArgs{
			Name:         ss2.name,
			Node1Address: ss2.node1Address.IP.String(),
			Node2Address: ss2.node2Address.IP.String(),
		})
	}

	return args
}

func export(input, output string, args interface{}) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()

	content, err := assets.ReadFile(path.Join("assets", input))
	if err != nil {
		return err
	}

	tmpl, err := template.New(input).Parse(string(content))
	if err != nil {
		return err
	}

	return tmpl.Execute(f, args)
}

func exportFile(input, output string, args interface{}) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := os.Open(input)
	if err != nil {
		return err
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	tmpl, err := template.New(input).Parse(string(content))
	if err != nil {
		return err
	}

	return tmpl.Execute(f, args)
}

func exportExecutableFile(input, output string, args interface{}) error {
	if err := export(input, output, args); err != nil {
		return err
	}

	f, err := os.Open(output)
	if err != nil {
		return err
	}
	defer f.Close()

	return f.Chmod(0755)
}

func copyFile(src, dist string) error {
	data, err := assets.ReadFile(path.Join("assets", src))
	if err != nil {
		return err
	}

	return os.WriteFile(dist, data, 0644)
}
