package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var httpClient = &well.HTTPClient{
	Client: &http.Client{},
}

// GraphQLQuery is GraphQL query to retrieve machine information from sabakan.
const GraphQLQuery = `
query rebootSearch($having: MachineParams = null,
                $notHaving: MachineParams = {
                  roles: ["boot"]
                }) {
  searchMachines(having: $having, notHaving: $notHaving) {
    spec {
		serial
		labels {
		  name
		  value
		}
      role
      ipv4
    }
  }
}
`

// State is the enum type for sabakan states.
type State string

// MachineParams is the query parameter type.
type MachineParams struct {
	Labels []struct {
		Name  string `json:"name" yaml:"name"`
		Value string `json:"value" yaml:"value"`
	} `json:"labels" yaml:"labels"`
	Racks               []int    `json:"racks" yaml:"racks"`
	Roles               []string `json:"roles" yaml:"roles"`
	States              []State  `json:"states" yaml:"states"`
	MinDaysBeforeRetire int      `json:"minDaysBeforeRetire" yaml:"minDaysBeforeRetire"`
}

// Machine represents a machine registered with sabakan.
type Machine struct {
	Spec struct {
		Serial string `json:"serial"`
		Labels []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"labels"`
		Role string   `json:"role"`
		IPv4 []string `json:"ipv4"`
	} `json:"spec"`
}

// QueryVariables represents the JSON object of the query variables.
type QueryVariables struct {
	Having    *MachineParams `json:"having"`
	NotHaving *MachineParams `json:"notHaving"`
}

var rebootWorkerCmd = &cobra.Command{
	Use:   "reboot-worker",
	Short: "Reboot all worker nodes.",
	Long:  `Reboot all worker nodes for their updates.`,

	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		url := neco.SabakanLocalEndpoint
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		machines, err := doQuery(ctx, url, nil, httpClient)
		if err != nil {
			log.ErrorExit(err)
		}
		driverVersion := getDriver()
		for _, m := range machines {
			well.Go(func(ctx context.Context) error {
				addr, err := lookupMachineBMCAddress(ctx, m.Spec.Serial)
				if err != nil {
					return err
				}
				return ipmiPower(ctx, "restart", driverVersion, addr)
			})
		}
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
		return
	},
}

func doQuery(ctx context.Context, url string, vars *QueryVariables, hc *well.HTTPClient) ([]Machine, error) {
	body := struct {
		Query     string          `json:"query"`
		Variables *QueryVariables `json:"variables,omitempty"`
	}{
		GraphQLQuery,
		vars,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sabakan returns %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Machines []Machine `json:"searchMachines"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("sabakan returns error: %v", result.Errors)
	}

	return result.Data.Machines, nil
}

func init() {
	rootCmd.AddCommand(rebootWorkerCmd)
}
