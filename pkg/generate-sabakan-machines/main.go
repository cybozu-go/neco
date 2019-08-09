package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	sabakan "github.com/cybozu-go/sabakan/v2"

	"github.com/cybozu-go/log"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: generate-sabakan-machines [stage0|tokyo0|osaka0] CSVFILE")
	os.Exit(2)
}

func main() {
	if len(os.Args) != 3 {
		usage()
	}

	var machineTypeBoot, machineTypeCS, machineTypeSS string
	switch os.Args[1] {
	case "stage0":
		machineTypeBoot = "r640-boot-1"
		machineTypeCS = "r640-cs-1"
		machineTypeSS = "r740xd-ss-2"
	case "tokyo0", "osaka0":
		machineTypeBoot = "r640-boot-2"
		machineTypeCS = "r640-cs-2"
		machineTypeSS = "r740xd-ss-2"
	default:
		log.ErrorExit(errors.New("specify valid datacenter"))
	}

	f, err := os.Open(os.Args[2])
	if err != nil {
		log.ErrorExit(err)
	}
	defer f.Close()

	ms := []sabakan.MachineSpec{}
	r := csv.NewReader(f)
	for l := 0; ; l++ {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.ErrorExit(err)
		}
		if l == 0 {
			continue
		}

		// Expected columns
		// 0: datacenter
		// 1: logical rack number
		// 2: physical rack number
		// 3: serial
		// 4: product
		// 5: role
		// 6: support date
		datacenter := record[0]
		rack := record[1]
		serial := record[3]
		product := record[4]
		role := record[5]

		var machineType string
		switch role {
		case "boot":
			machineType = machineTypeBoot
		case "cs":
			machineType = machineTypeCS
		case "ss":
			machineType = machineTypeSS
		default:
			log.ErrorExit(errors.New("unknown role " + role))
		}
		if len(datacenter) == 0 || len(rack) == 0 || len(serial) == 0 || len(product) == 0 || len(role) == 0 {
			log.ErrorExit(
				fmt.Errorf("some colmuns are missing in csv, datacenter: %s, rack: %s, serial: %s, product: %s, role: %s",
					datacenter, rack, serial, product, role))
		}

		rackInt, err := strconv.ParseInt(rack, 10, 64)
		if err != nil {
			log.ErrorExit(err)
		}

		m := sabakan.MachineSpec{
			Serial: serial,
			Labels: map[string]string{
				"machine-type": machineType,
				"datacenter":   datacenter,
				"product":      product,
			},
			Rack: uint(rackInt),
			Role: role,
			BMC: sabakan.MachineBMC{
				Type: "IPMI-2.0",
			},
		}

		ms = append(ms, m)
	}

	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	err = e.Encode(ms)
	if err != nil {
		log.ErrorExit(err)
	}
	os.Exit(0)
}
