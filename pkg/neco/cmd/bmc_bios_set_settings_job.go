package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/stmcginnis/gofish"
)

func setBiosSettingsJob(client *gofish.APIClient) (string, error) {
	payload := map[string]string{
		"TargetSettingsURI": "/redfish/v1/Systems/System.Embedded.1/Bios/Settings",
	}
	resp, err := client.Post(jobEndpoint, payload)
	if err != nil {
		return "", err
	}
	jobURL, err := resp.Location()
	if err != nil {
		return "", err
	}
	split := strings.Split(string(jobURL.Path), "/")
	return split[len(split)-1], nil
}

var bmcBiosSetSettingsJobCmd = &cobra.Command{
	Use:   "settings-job SERIAL|IP",
	Short: "set bios settings job",
	Long:  `set bios settings job`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			client, err := getRedfishClient(ctx, args[0])
			if err != nil {
				return err
			}
			defer client.Logout()

			jobID, err := setBiosSettingsJob(client)
			if err != nil {
				return err
			}
			fmt.Println(jobID)
			return nil
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	bmcBiosSetCmd.AddCommand(bmcBiosSetSettingsJobCmd)
}
