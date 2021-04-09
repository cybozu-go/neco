package cmd

import (
	"context"
	"encoding/json"
	"io"
	"path"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/common"
)

const jobEndpoint = "/redfish/v1/Managers/iDRAC.Embedded.1/Jobs"

func getJobIDs(client *gofish.APIClient) ([]string, error) {
	resp, err := client.Get(jobEndpoint)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jobLinks common.LinksCollection
	err = json.Unmarshal(body, &jobLinks)
	if err != nil {
		return nil, err
	}
	jobIDs := make([]string, jobLinks.Count)
	for i, link := range jobLinks.Members {
		split := strings.Split(string(link), "/")
		jobIDs[i] = split[len(split)-1]
	}
	return jobIDs, nil
}

func getJob(client *gofish.APIClient, jobID string) (interface{}, error) {
	endpoint := path.Join(jobEndpoint, jobID)
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var job interface{}
	err = json.Unmarshal(body, &job)
	if err != nil {
		return nil, err
	}
	return job, nil
}

var bmcJobGetCmd = &cobra.Command{
	Use:   "get SERIAL|IP [JOB_ID...]",
	Short: "get idrac job of a machine",
	Long: `Control power of a machine using Redfish API.

	SERIAL is the serial number of the machine.
	IP is one of the IP addresses owned by the machine.`,

	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			client, err := getRedfishClient(ctx, args[0])
			if err != nil {
				return err
			}
			defer client.Logout()

			jobIDs := args[1:]

			if len(jobIDs) == 1 {
				job, err := getJob(client, args[1])
				if err != nil {
					return err
				}
				e := json.NewEncoder(cmd.OutOrStdout())
				e.SetIndent("", "  ")
				return e.Encode(job)
			}

			if len(jobIDs) == 0 {
				all, err := getJobIDs(client)
				if err != nil {
					return err
				}
				jobIDs = all
			}

			jobList := make([]interface{}, len(jobIDs))
			for index, id := range jobIDs {
				job, err := getJob(client, id)
				if err != nil {
					return err
				}
				jobList[index] = job
			}
			e := json.NewEncoder(cmd.OutOrStdout())
			e.SetIndent("", "  ")
			return e.Encode(jobList)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	bmcJobCmd.AddCommand(bmcJobGetCmd)
}
