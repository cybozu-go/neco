package cmd

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/cybozu-go/log"
	"github.com/spf13/cobra"
)

const serfTagUptime = "uptime"

// serfMember is copied from type Member https://godoc.org/github.com/hashicorp/serf/cmd/serf/command#Member
// to prevent much vendoring
type serfMember struct {
	Name   string            `json:"name"`
	Addr   string            `json:"addr"`
	Port   uint16            `json:"port"`
	Tags   map[string]string `json:"tags"`
	Status string            `json:"status"`
	Proto  map[string]uint8  `json:"protocol"`
	// contains filtered or unexported fields
}

// serfMemberContainer is copied from type MemberContainer https://godoc.org/github.com/hashicorp/serf/cmd/serf/command#MemberContainer
// to prevent much vendoring
type serfMemberContainer struct {
	Members []serfMember `json:"members"`
}

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

	var oldUptime string
	member, err := getSerfMemberBySerial(machine.Spec.Serial)
	if err != nil {
		log.Error("failed to get serf member", map[string]interface{}{
			"serial_or_ip": target,
			log.FnError:    err,
		})
		return err
	}
	if member != nil {
		oldUptime = member.Tags[serfTagUptime]
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

	for {
		time.Sleep(1 * time.Second)

		member, err := getSerfMemberBySerial(machine.Spec.Serial)
		if err != nil {
			log.Error("failed to get serf member", map[string]interface{}{
				"serial_or_ip": target,
				log.FnError:    err,
			})
			return err
		}

		if member == nil || member.Tags[serfTagUptime] == "" {
			continue
		}
		if member.Tags[serfTagUptime] != oldUptime {
			return nil
		}
	}
}

func getSerfMemberBySerial(serial string) (*serfMember, error) {
	cmd := exec.Command("serf", "members", "-format", "json", "-tag", "serial="+serial)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	container := new(serfMemberContainer)
	err = json.Unmarshal(output, container)
	if err != nil {
		return nil, err
	}

	if len(container.Members) == 0 {
		return nil, nil
	}
	return &container.Members[0], nil
}

func init() {
	rootCmd.AddCommand(rebootAndWaitCmd)
}
