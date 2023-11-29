package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/sabakan/v3"
	"github.com/spf13/cobra"
)

const serfTagUptime = "uptime"
const serfTagUptimeFormat = "2006-01-02 15:04:05"

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

var rebootCheckCmd = &cobra.Command{
	Use:   "reboot-check SERIAL_OR_IP UNIXTIME",
	Short: "check machine's (re)boot-up",
	Long:  `Check (re)boot-up of a machine having SERIAL or IP address after the UNIXTIME.`,

	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		target := args[0]
		timestamp := args[1]
		return rebootCheckMain(target, timestamp)
	},
}

func rebootCheckMain(target, timestamp string) error {
	unixtime, err := strconv.Atoi(timestamp)
	if err != nil {
		log.Error("failed parse timestamp", map[string]interface{}{
			"timestamp": timestamp,
			log.FnError: err,
		})
		return err
	}
	tm := time.Unix(int64(unixtime), 0)

	rebooted, err := rebootCheck(target, tm, false)
	if err != nil {
		return err
	}

	if rebooted {
		fmt.Println("true")
	} else {
		fmt.Println("false")
	}

	return nil
}

func rebootCheck(target string, timestamp time.Time, wait bool) (bool, error) {
	machine, err := lookupMachine(context.Background(), target)
	if err != nil {
		log.Error("failed to lookup serial or IP address", map[string]interface{}{
			"serial_or_ip": target,
			log.FnError:    err,
		})
		return false, err
	}

	for {
		member, err := getSerfMemberBySerial(machine.Spec.Serial)
		if err != nil {
			log.Error("failed to get serf member", map[string]interface{}{
				"serial_or_ip": target,
				log.FnError:    err,
			})
			return false, err
		}

		if member != nil && member.Tags[serfTagUptime] != "" {
			uptime, err := time.Parse(serfTagUptimeFormat, member.Tags[serfTagUptime])
			if err != nil {
				log.Error("failed to parse uptime from serf member", map[string]interface{}{
					"serial_or_ip": target,
					"uptime":       member.Tags[serfTagUptime],
					log.FnError:    err,
				})
				return false, err
			}

			if uptime.After(timestamp) {
				break
			}
		}

		if !wait {
			return false, nil
		}
		time.Sleep(1 * time.Second)
	}

	for {
		machine, err := lookupMachine(context.Background(), target)
		if err != nil {
			log.Error("failed to lookup serial or IP address", map[string]interface{}{
				"serial_or_ip": target,
				log.FnError:    err,
			})
			return false, err
		}

		if machine.Status.State == sabakan.StateHealthy {
			break
		}

		if !wait {
			return false, nil
		}
		time.Sleep(1 * time.Second)
	}

	return true, nil
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
	rootCmd.AddCommand(rebootCheckCmd)
}
