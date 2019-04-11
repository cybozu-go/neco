package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/tcnksm/go-input"
)

var httpClient = &well.HTTPClient{
	Client: &http.Client{},
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
		BMC  BMC      `json:"bmc"`
	} `json:"spec"`
}

// BMC contains a machine's BMC information.
type BMC struct {
	IPv4 string `json:"ipv4"`
}

// graphQLQuery is GraphQL query to retrieve machine information from sabakan.
const graphQLQuery = `
query rebootSearch($having: MachineParams = null,
					$notHaving: MachineParams = {
						roles: ["boot"]
					}) {
	searchMachines(having: $having, notHaving: $notHaving) {
		spec {
			serial
			bmc {
				ipv4
			}
		}
	}
}
`

var rebootWorkerCmd = &cobra.Command{
	Use:   "reboot-worker",
	Short: "Reboot all worker nodes.",
	Long:  `Reboot all worker nodes for their updates.`,

	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		graphQLEndpoint := neco.SabakanLocalEndpoint + "/graphql"
		fmt.Println("WARNING: this command reboots all servers other than boot servers and will cause a system down.")
		ans, err := askYorN("Continue?")
		if err != nil {
			log.ErrorExit(err)
		}
		if !ans {
			return
		}
		fmt.Println("WARNING: rebooting starts immediately after this question.")
		ans, err = askYorN("Continue?")
		if err != nil {
			log.ErrorExit(err)
		}
		if !ans {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		machines, err := doQuery(ctx, graphQLEndpoint, graphQLQuery, httpClient)
		if err != nil {
			log.ErrorExit(err)
		}
		driverVersion := getDriver()
		for _, m := range machines {
			addr := m.Spec.BMC.IPv4
			if addr == "" {
				log.ErrorExit(errors.New(m.Spec.Serial + "'s BMC IPAddress not found"))
			}
			err := ipmiPower(context.Background(), "restart", driverVersion, addr)
			if err != nil {
				log.ErrorExit(err)
			}
		}
		return
	},
}

func askYorN(query string) (bool, error) {
	ans, err := input.DefaultUI().Ask(query+" [y/N]", &input.Options{
		Default:     "N",
		HideDefault: true,
		HideOrder:   true,
	})
	if err != nil {
		return false, err
	}
	switch ans {
	case "y", "Y", "yes", "YES":
		return true, nil
	}
	return false, nil
}

func doQuery(ctx context.Context, url string, query string, hc *well.HTTPClient) ([]Machine, error) {
	body := struct {
		Query string `json:"query"`
	}{
		query,
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
