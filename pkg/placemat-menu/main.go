package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cybozu-go/neco/menu"
)

var (
	flagConfig = flag.String("f", "", "Template file for placemat-menu")
	flagOutDir = flag.String("o", ".", "Directory for output files")

	staticFiles = []string{"squid.conf", "chrony.conf"}
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	sabakanDir := filepath.Join(*flagOutDir, "sabakan")
	err := os.MkdirAll(sabakanDir, 0755)
	if err != nil {
		return err
	}

	configFile, err := filepath.Abs(*flagConfig)
	if err != nil {
		return err
	}
	dir := filepath.Dir(configFile)

	f, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer f.Close()

	m, err := menu.ReadYAML(dir, bufio.NewReader(f))
	if err != nil {
		return err
	}
	ta, err := menu.ToTemplateArgs(m)
	if err != nil {
		return err
	}

	clusterFile, err := os.Create(filepath.Join(*flagOutDir, "cluster.yml"))
	if err != nil {
		return err
	}
	defer clusterFile.Close()
	err = menu.ExportCluster(clusterFile, ta)
	if err != nil {
		return err
	}

	err = export("setup-iptables", "setup-iptables", true, ta.Network.Exposed.Global)
	if err != nil {
		return err
	}

	err = export("setup-default-gateway", "setup-default-gateway-operation", true, ta.Core.OperationAddress)
	if err != nil {
		return err
	}
	err = export("setup-default-gateway", "setup-default-gateway-external", true, ta.Core.ExternalAddress)
	if err != nil {
		return err
	}

	err = export("bird_core.conf", "bird_core.conf", false, ta)
	if err != nil {
		return err
	}
	for spineIdx := range ta.Spines {
		err = export("bird_spine.conf",
			fmt.Sprintf("bird_spine%d.conf", spineIdx+1),
			false,
			menu.BIRDSpineTemplateArgs{Args: *ta, SpineIdx: spineIdx})
		if err != nil {
			return err
		}
	}

	networkFile, err := os.Create(filepath.Join(*flagOutDir, "network.yml"))
	if err != nil {
		return err
	}
	defer networkFile.Close()
	err = menu.ExportEmptyNetworkConfig(networkFile)
	if err != nil {
		return err
	}

	for rackIdx, rack := range ta.Racks {
		if ta.Boot.CloudInitTemplate != "" {
			arg := struct {
				Name string
				Rack menu.Rack
			}{
				fmt.Sprintf("boot-%d", rack.Index),
				rack,
			}
			name := ta.Boot.CloudInitTemplate
			if !filepath.IsAbs(name) {
				name = filepath.Join(dir, name)
			}
			err := exportFile(name, fmt.Sprintf("seed_boot-%d.yml", rack.Index), arg)
			if err != nil {
				return err
			}
		}

		if ta.CS.CloudInitTemplate != "" {
			for _, cs := range rack.CSList {
				arg := struct {
					Name string
					Rack menu.Rack
				}{
					fmt.Sprintf("%s-%s", rack.Name, cs.Name),
					rack,
				}
				name := ta.CS.CloudInitTemplate
				if !filepath.IsAbs(name) {
					name = filepath.Join(dir, name)
				}
				err := exportFile(name, fmt.Sprintf("seed_%s-%s.yml", rack.Name, cs.Name), arg)
				if err != nil {
					return err
				}
			}
		}

		if ta.SS.CloudInitTemplate != "" {
			for _, ss := range rack.SSList {
				arg := struct {
					Name string
					Rack menu.Rack
				}{
					fmt.Sprintf("%s-%s", rack.Name, ss.Name),
					rack,
				}
				name := ta.SS.CloudInitTemplate
				if !filepath.IsAbs(name) {
					name = filepath.Join(dir, name)
				}
				err := exportFile(name, fmt.Sprintf("seed_%s-%s.yml", rack.Name, ss.Name), arg)
				if err != nil {
					return err
				}
			}
		}

		err = export("bird_rack-tor1.conf",
			fmt.Sprintf("bird_rack%d-tor1.conf", rackIdx),
			false,
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			return err
		}

		err = export("bird_rack-tor2.conf",
			fmt.Sprintf("bird_rack%d-tor2.conf", rackIdx),
			false,
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			return err
		}
	}

	err = menu.ExportSabakanData(sabakanDir, m, ta)
	if err != nil {
		return err
	}

	return copyStatics(staticFiles, *flagOutDir)
}

func exportFile(input string, output string, args interface{}) error {
	f, err := os.Create(filepath.Join(*flagOutDir, output))
	if err != nil {
		return err
	}
	defer f.Close()

	templateFile, err := os.Open(input)
	if err != nil {
		return err
	}
	content, err := ioutil.ReadAll(templateFile)
	if err != nil {
		return err
	}

	tmpl, err := template.New(input).Parse(string(content))
	if err != nil {
		panic(err)
	}
	return tmpl.Execute(f, args)
}

func export(name, output string, executable bool, args interface{}) error {
	f, err := os.Create(filepath.Join(*flagOutDir, output))
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl, err := menu.GetAssetTemplate(name)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(f, args)
	if err != nil {
		return err
	}

	if !executable {
		return nil
	}

	return f.Chmod(0755)
}

func copyStatics(names []string, outputDirName string) error {
	for _, name := range names {
		err := copyStatic(name, outputDirName)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyStatic(name, outputDirName string) error {
	data, ok := menu.Assets[name]
	if !ok {
		panic("no such asset: " + name)
	}

	return ioutil.WriteFile(filepath.Join(outputDirName, name), []byte(data), 0644)
}
