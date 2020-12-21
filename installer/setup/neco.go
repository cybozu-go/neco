package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/cybozu-go/neco"
)

func dumpNecoFiles(lrn int) error {
	fmt.Fprintln(os.Stderr, "Creating /etc/neco files...")

	if err := os.MkdirAll(neco.NecoDir, 0755); err != nil {
		return fmt.Errorf("failed to mkdir %s: %w", neco.NecoDir, err)
	}

	if err := ioutil.WriteFile(neco.RackFile, []byte(fmt.Sprintf("%d\n", lrn)), 0644); err != nil {
		return fmt.Errorf("failed to create %s: %w", neco.RackFile, err)
	}

	if err := ioutil.WriteFile(neco.ClusterFile, []byte(config.cluster.Name+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to create %s: %w", neco.ClusterFile, err)
	}

	_, n, _ := net.ParseCIDR(config.cluster.BMC)
	prefixLen, _ := n.Mask.Size()
	sabaconf := map[string]interface{}{
		"max-nodes-in-rack":       28,
		"node-ipv4-pool":          "10.69.0.0/20",
		"node-ipv4-range-size":    6,
		"node-ipv4-range-mask":    26,
		"node-index-offset":       3,
		"node-ip-per-node":        3,
		"node-gateway-offset":     1,
		"bmc-ipv4-pool":           config.cluster.BMC,
		"bmc-ipv4-offset":         "0.0.1.0",
		"bmc-ipv4-range-size":     5,
		"bmc-ipv4-range-mask":     prefixLen,
		"bmc-ipv4-gateway-offset": 1,
	}
	data, err := json.Marshal(sabaconf)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(neco.SabakanIPAMFile, data, 0644); err != nil {
		return fmt.Errorf("failed to create%s: %w", neco.SabakanIPAMFile, err)
	}
	return nil
}
