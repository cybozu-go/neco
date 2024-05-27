package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "generate-sabakan-machines CSV",
	SilenceUsage: true,
	Short:        "generate machines.json",
	Long: `Generate machines.json from the CSV file for use with the 'sabactl create' command.
Example:
  generate-sabakan-machines input.csv --machine-type-boot=r6525-boot-1 --machine-type-cs=r6525-cs-1 --machine-type-ss=r7525-ss-1
	`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.Open(args[0])
		if err != nil {
			return err
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
				return err
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
				machineType = *machineTypeBoot
			case "cs":
				machineType = *machineTypeCS
			case "ss":
				machineType = *machineTypeSS
			default:
				return errors.New("unknown role " + role)
			}
			if len(datacenter) == 0 || len(rackString) == 0 || len(serial) == 0 ||
				len(product) == 0 || len(role) == 0 || len(supportDateString) == 0 {
				return fmt.Errorf("some colmuns are missing in csv, datacenter: %s, rack: %s, serial: %s, product: %s, role: %s",
					datacenter, rackString, serial, product, role)
			}

			rack, err := strconv.Atoi(rackString)
			if err != nil {
				return err
			}

			supportDate, err := time.Parse("2006/1/2", supportDateString)
			if err != nil {
				return err
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
					"type": *bmcType,
				},
			}
			ms = append(ms, m)
		}

		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		err = e.Encode(ms)
		if err != nil {
			return err
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var (
	machineTypeBoot *string
	machineTypeCS   *string
	machineTypeSS   *string
	bmcType         *string
)

func init() {
	machineTypeBoot = rootCmd.Flags().StringP("machine-type-boot", "b", "", "The machine-type name of Boot")
	machineTypeCS = rootCmd.Flags().StringP("machine-type-cs", "c", "", "The machine-type name of CS")
	machineTypeSS = rootCmd.Flags().StringP("machine-type-ss", "s", "", "The machine-type name of SS")
	bmcType = rootCmd.Flags().String("bmc-type", "iDRAC", "The name of BMC")

	rootCmd.MarkFlagRequired("machine-type-boot")
	rootCmd.MarkFlagRequired("machine-type-cs")
	rootCmd.MarkFlagRequired("machine-type-ss")
}
