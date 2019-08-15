package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

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

	var machineTypeBoot, machineTypeCS, machineTypeSS, bmcType string
	switch os.Args[1] {
	case "stage0":
		machineTypeBoot = "r640-boot-1"
		machineTypeCS = "r640-cs-1"
		machineTypeSS = "r740xd-ss-2"
		bmcType = "iDRAC"
	case "tokyo0", "osaka0":
		machineTypeBoot = "r640-boot-2"
		machineTypeCS = "r640-cs-2"
		machineTypeSS = "r740xd-ss-2"
		bmcType = "iDRAC"
	default:
		log.ErrorExit(errors.New("specify valid datacenter"))
	}

	f, err := os.Open(os.Args[2])
	if err != nil {
		log.ErrorExit(err)
	}
	defer f.Close()

	ms := []map[string]interface{}{}
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
		rackString := record[1]
		serial := record[3]
		product := record[4]
		role := record[5]
		supportDateString := record[6]

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
		if len(datacenter) == 0 || len(rackString) == 0 || len(serial) == 0 ||
			len(product) == 0 || len(role) == 0 || len(supportDateString) == 0 {
			log.ErrorExit(
				fmt.Errorf("some colmuns are missing in csv, datacenter: %s, rack: %s, serial: %s, product: %s, role: %s",
					datacenter, rackString, serial, product, role))
		}

		rack, err := strconv.ParseInt(rackString, 30, 64)
		if err != nil {
			log.ErrorExit(err)
		}

		supportDate, err := time.Parse("2006/1/02", supportDateString)
		if err != nil {
			log.ErrorExit(err)
		}
		// retireDate = support date + 5 years
		retireDate := supportDate.Add(time.Hour * 24 * 365 * 5)

		m := map[string]interface{}{
			"serial": serial,
			"labels": map[string]string{
				"datacenter":   datacenter,
				"machine-type": machineType,
				"product":      product,
			},
			"rack":        rack,
			"role":        role,
			"retire-date": retireDate.Format(time.RFC3339),
			"bmc": map[string]string{
				"type": bmcType,
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
