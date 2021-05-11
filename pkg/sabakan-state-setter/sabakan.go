package sss

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/sabakan/v2"
	sabac "github.com/cybozu-go/sabakan/v2/client"
	"github.com/cybozu-go/sabakan/v2/gql/graph/model"
	"github.com/vektah/gqlparser/gqlerror"
)

const machineTypeLabelName = "machine-type"

// SabakanClientWrapper is interface of the sabakan client for sbakan-state-setter
type SabakanClientWrapper interface {
	GetAllMachines(ctx context.Context) ([]*machine, error)
	GetRetiredMachines(ctx context.Context) ([]*machine, error)
	UpdateSabakanState(ctx context.Context, serial string, state sabakan.MachineState) error
	CryptsDelete(ctx context.Context, serial string) error
}

// machine is a subset of sabakan.machine for sabakan-state-setter.
// This consists of the fields which sabakan-state-setter needs.
type machine struct {
	Serial   string
	Type     string
	IPv4Addr string
	State    sabakan.MachineState
}

// searchMachineResponse is a machine struct of response from the sabakan
type searchMachineResponse struct {
	SearchMachines []searchMachine `json:"searchMachines"`
}

type searchMachine struct {
	Spec   spec   `json:"spec"`
	Status status `json:"status"`
}

type spec struct {
	Serial string   `json:"serial"`
	Labels []label  `json:"labels"`
	IPv4   []string `json:"ipv4"`
}

type label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type status struct {
	State string `json:"state"`
}

// QueryVariables represents the JSON object of the query variables.
type graphQLVariables struct {
	Having *model.MachineParams `json:"having"`
}

type graphQLRequest struct {
	Query     string           `json:"query"`
	Variables graphQLVariables `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage  `json:"data"`
	Errors []gqlerror.Error `json:"errors,omitempty"`
}

type sabacWrapper struct {
	httpClient    *http.Client
	sabakanClient *sabac.Client
	gqlEndpoint   string
}

func toMachineState(str string) sabakan.MachineState {
	switch str {
	case "UNINITIALIZED":
		return sabakan.StateUninitialized
	case "HEALTHY":
		return sabakan.StateHealthy
	case "UNHEALTHY":
		return sabakan.StateUnhealthy
	case "UNREACHABLE":
		return sabakan.StateUnreachable
	case "UPDATING":
		return sabakan.StateUpdating
	case "RETIRING":
		return sabakan.StateRetiring
	case "RETIRED":
		return sabakan.StateRetired
	}
	return ""
}

func findLabelValue(labels []label, name string) string {
	for _, l := range labels {
		if l.Name == name {
			return l.Value
		}
	}
	return ""
}

func newSabakanGQLClient(address string) (SabakanClientWrapper, error) {
	httpClient := ext.LocalHTTPClient()
	sabakanClient, err := sabac.NewClient(address, httpClient)
	if err != nil {
		return nil, err
	}
	gqlEndpoint, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	gqlEndpoint.Path = path.Join(gqlEndpoint.Path, "/graphql")
	return &sabacWrapper{
		httpClient:    httpClient,
		sabakanClient: sabakanClient,
		gqlEndpoint:   gqlEndpoint.String(),
	}, nil
}

func (c *sabacWrapper) requestGQL(ctx context.Context, greq graphQLRequest) ([]byte, error) {
	data, err := json.Marshal(greq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.gqlEndpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	// gqlgen 0.9+ requires application/json content-type header.
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gresp graphQLResponse
	err = json.NewDecoder(resp.Body).Decode(&gresp)
	if err != nil {
		return nil, err
	}

	if len(gresp.Errors) > 0 {
		return nil, &gresp.Errors[0]
	}
	return []byte(gresp.Data), nil
}

func (c *sabacWrapper) getMachines(ctx context.Context, having *model.MachineParams) ([]*machine, error) {
	greq := graphQLRequest{
		Query: `
query search($having: MachineParams) {
  searchMachines(having: $having, notHaving: null) {
    spec {
      serial
      labels {
        name
        value
      }
      ipv4
    }
    status {
      state
    }
  }
}`,
		Variables: graphQLVariables{
			Having: having,
		},
	}
	gdata, err := c.requestGQL(ctx, greq)
	if err != nil {
		return nil, err
	}

	resp := new(searchMachineResponse)
	err = json.Unmarshal(gdata, resp)
	if err != nil {
		return nil, err
	}

	ret := make([]*machine, len(resp.SearchMachines))
	for i, m := range resp.SearchMachines {
		ret[i] = &machine{
			Serial:   m.Spec.Serial,
			Type:     findLabelValue(m.Spec.Labels, machineTypeLabelName),
			IPv4Addr: m.Spec.IPv4[0],
			State:    toMachineState(m.Status.State),
		}
	}
	return ret, nil
}

// GetSabakanMachines returns all machines
func (c *sabacWrapper) GetAllMachines(ctx context.Context) ([]*machine, error) {
	return c.getMachines(ctx, nil)
}

// GetRetiredMachines returns retired machines
func (c *sabacWrapper) GetRetiredMachines(ctx context.Context) ([]*machine, error) {
	return c.getMachines(ctx, &model.MachineParams{
		States: []sabakan.MachineState{sabakan.StateRetired},
	})
}

// UpdateSabakanState updates given machine's state
func (c *sabacWrapper) UpdateSabakanState(ctx context.Context, serial string, state sabakan.MachineState) error {
	if !state.IsValid() {
		return fmt.Errorf("invalid state: %s", state)
	}
	greq := graphQLRequest{
		Query: fmt.Sprintf(`mutation {
  setMachineState(serial: "%s", state: %s) {
    state
  }
}`, serial, state.GQLEnum()),
	}

	_, err := c.requestGQL(ctx, greq)
	return err
}

// CryptsDelete is wapper function of sabakan.Client's CryptsDelete().
func (c *sabacWrapper) CryptsDelete(ctx context.Context, serial string) error {
	return c.sabakanClient.CryptsDelete(ctx, serial)
}
