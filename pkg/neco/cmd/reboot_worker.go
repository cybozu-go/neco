package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/tcnksm/go-input"
	"sigs.k8s.io/yaml"
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
			ipv4
			bmc {
				ipv4
			}
		}
	}
}
`

// ckeCluster is part of cke.Cluster in github.com/cybozu-go/cke
type ckeCluster struct {
	Nodes []*ckeNode `yaml:"nodes"`
}

// ckeNode is part of cke.Node in github.com/cybozu-go/cke
type ckeNode struct {
	Address string            `yaml:"address"`
	Labels  map[string]string `yaml:"labels"`
}

var rebootWorkerCmd = &cobra.Command{
	Use:   "reboot-worker",
	Short: "reboot all worker nodes",
	Long: `Reboot all worker nodes for their updates.

This uses CKE's function of graceful reboot for the nodes used by CKE.
As for the other nodes, this reboots them immediately.`,

	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("WARNING: this command reboots all servers other than boot servers and may cause system instability.")
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

		err = rebootMain()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func rebootMain() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	graphQLEndpoint := neco.SabakanLocalEndpoint + "/graphql"
	machines, err := doQuery(ctx, graphQLEndpoint, graphQLQuery, httpClient)
	if err != nil {
		return err
	}

	comm := exec.Command(neco.CKECLIBin, "cluster", "get")
	comm.Stderr = os.Stderr
	output, err := comm.Output()
	if err != nil {
		return err
	}
	cluster := new(ckeCluster)
	err = yaml.Unmarshal(output, cluster)
	if err != nil {
		return err
	}

	var ss, nonss []string
	for _, node := range cluster.Nodes {
		if node.Labels["cke.cybozu.com/role"] == "ss" {
			ss = append(ss, node.Address)
			continue
		}

		nonss = append(nonss, node.Address)
	}

	for {
		addresses := make([]string, 0, 2)
		if len(ss) > 0 {
			addresses = append(addresses, ss[0])
			ss = ss[1:]
		}
		if len(nonss) > 0 {
			addresses = append(addresses, nonss[0])
			nonss = nonss[1:]
		}
		if len(addresses) == 0 {
			break
		}

		log.Info("adding node to CKE reboot queue", map[string]interface{}{
			"nodes": addresses,
		})
		comm := exec.Command(neco.CKECLIBin, "reboot-queue", "add", "-")
		comm.Stdin = strings.NewReader(strings.Join(addresses, "\n"))
		comm.Stdout = os.Stdout
		comm.Stderr = os.Stderr
		err := comm.Run()
		if err != nil {
			return err
		}
	}

	ckeNodeAddrs := make(map[string]bool, len(cluster.Nodes))
	for _, node := range cluster.Nodes {
		ckeNodeAddrs[node.Address] = true
	}
	for _, m := range machines {
		if len(m.Spec.IPv4) == 0 {
			log.Warn("IP addresses not found; skipping", map[string]interface{}{
				"serial": m.Spec.Serial,
			})
			continue
		}
		if ckeNodeAddrs[m.Spec.IPv4[0]] {
			continue
		}
		addr := m.Spec.BMC.IPv4
		if addr == "" {
			log.Warn("BMC IP address not found; skipping", map[string]interface{}{
				"serial": m.Spec.Serial,
				"node":   m.Spec.IPv4[0],
			})
			continue
		}
		log.Info("rebooting node via IPMI", map[string]interface{}{
			"serial": m.Spec.Serial,
			"node":   m.Spec.IPv4[0],
			"bmc":    m.Spec.BMC.IPv4,
		})
		err := power(context.Background(), "restart", addr)
		if err != nil {
			return err
		}
	}

	return nil
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

	// gqlgen 0.9+ requires application/json content-type header.
	req.Header.Set("Content-Type", "application/json")

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
