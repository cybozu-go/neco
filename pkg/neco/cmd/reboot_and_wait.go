package cmd

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/spf13/cobra"
)

var rebootAndWaitCmd = &cobra.Command{
	Use:   "reboot-and-wait SERIAL_OR_IP",
	Short: "reboot a machine and wait for its boot-up",
	Long:  `Reboot a machine having SERIAL or IP address, and wait for its boot-up.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		target := args[0]
		if target == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				log.Error("failed to read node object", map[string]interface{}{
					log.FnError: err,
				})
				return err
			}
			var node struct {
				Address string `json:"address"`
			}
			err = json.Unmarshal(data, &node)
			if err != nil {
				log.Error("invalid node object", map[string]interface{}{
					"stdin":     string(data),
					log.FnError: err,
				})
				return err
			}
			target = node.Address
		}
		return rebootAndWaitMain(target)
	},
}

func rebootAndWaitMain(target string) error {
	machine, err := lookupMachine(context.Background(), target)
	if err != nil {
		log.Error("failed to lookup serial or IP address", map[string]interface{}{
			"serial_or_ip": target,
			log.FnError:    err,
		})
		return err
	}

	var oldUptime time.Time
	member, err := getSerfMemberBySerial(machine.Spec.Serial)
	if err != nil {
		log.Error("failed to get serf member", map[string]interface{}{
			"serial_or_ip": target,
			log.FnError:    err,
		})
		return err
	}
	if member != nil && member.Tags[serfTagUptime] != "" {
		oldUptime, err = time.Parse(serfTagUptimeFormat, member.Tags[serfTagUptime])
		if err != nil {
			log.Error("failed to parse uptime from serf member", map[string]interface{}{
				"serial_or_ip": target,
				"uptime":       member.Tags[serfTagUptime],
				log.FnError:    err,
			})
			return err
		}
	}

	log.Info("rebooting a machine", map[string]interface{}{
		"serial_or_ip": target,
	})
	err = power(context.Background(), "restart", machine.Spec.BMC.IPv4)
	if err != nil {
		log.Error("failed to reboot via IPMI", map[string]interface{}{
			"serial_or_ip": target,
			log.FnError:    err,
		})
		return err
	}

	// sleep for a while to ignore a delayed change of uptime occurred before my reboot, if any
	time.Sleep(60 * time.Second)

	_, err = rebootCheck(target, oldUptime, true)
	return err
}

func init() {
	rootCmd.AddCommand(rebootAndWaitCmd)
}
