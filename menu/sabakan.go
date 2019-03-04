package menu

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cybozu-go/sabakan/v2"
)

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

func sabakanMachine(serial string, rack int, role string) sabakan.MachineSpec {
	return sabakan.MachineSpec{
		Serial: serial,
		Labels: map[string]string{
			"product":    "vm",
			"datacenter": "dc1",
		},
		Rack: uint(rack),
		Role: role,
		BMC: sabakan.MachineBMC{
			Type: "IPMI-2.0",
		},
	}
}

func exportMachinesJSON(dst string, ta *TemplateArgs) error {
	var ms []sabakan.MachineSpec

	for _, rack := range ta.Racks {
		ms = append(ms, sabakanMachine(rack.BootNode.Serial, rack.Index, "boot"))

		for _, cs := range rack.CSList {
			ms = append(ms, sabakanMachine(cs.Serial, rack.Index, "worker"))
		}
		for _, ss := range rack.SSList {
			ms = append(ms, sabakanMachine(ss.Serial, rack.Index, "worker"))
		}
	}

	return exportJSON(dst, ms)
}

// ExportSabakanData exports configuration files for sabakan
func ExportSabakanData(dir string, m *Menu, ta *TemplateArgs) error {
	return exportMachinesJSON(filepath.Join(dir, "machines.json"), ta)
}
