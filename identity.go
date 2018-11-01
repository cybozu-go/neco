package neco

import (
	"bytes"
	"io/ioutil"
	"strconv"
)

var rackFile = "/etc/neco/rack"
var clusterFile = "/etc/neco/cluster"

// MyLRN returns logical rack number of own node
func MyLRN() (int, error) {
	data, err := ioutil.ReadFile(rackFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(bytes.TrimSpace(data)))
}

// MyCluster returns cluster name of own node
func MyCluster() (string, error) {
	data, err := ioutil.ReadFile(clusterFile)
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(data)), nil
}
