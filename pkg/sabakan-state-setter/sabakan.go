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
	"github.com/vektah/gqlparser/gqlerror"
)

const machineTypeLabelName = "machine-type"

// SabakanGQLClient is interface of the sabakan client of GraphQL
type SabakanGQLClient interface {
	GetSabakanMachines(ctx context.Context) ([]*machine, error)
	UpdateSabakanState(ctx context.Context, serial string, state sabakan.MachineState) error
}

// machine is a subset of sabakan.machine for sabakan-state-setter.
// This consists of the fields which sabakan-state-setter needs.
type machine struct {
	Serial   string
	Type     string
	IPv4Addr string
	BMCAddr  string
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
	BMC    bmc      `json:"bmc"`
}

type label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type bmc struct {
	IPv4 string `json:"ipv4"`
}

type status struct {
	State string `json:"state"`
}

type graphQLRequest struct {
	Query string `json:"query"`
}

type graphQLResponse struct {
	Data   json.RawMessage  `json:"data"`
	Errors []gqlerror.Error `json:"errors,omitempty"`
}

type gqlClient struct {
	httpClient *http.Client
	endpoint   string
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

func newSabakanGQLClient(address string) (SabakanGQLClient, error) {
	baseURL, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	baseURL.Path = path.Join(baseURL.Path, "/graphql")
	sabakanEndpoint := baseURL.String()
	return &gqlClient{ext.LocalHTTPClient(), sabakanEndpoint}, nil
}

func (g *gqlClient) requestGQL(ctx context.Context, greq graphQLRequest) ([]byte, error) {
	data, err := json.Marshal(greq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, g.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	// gqlgen 0.9+ requires application/json content-type header.
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
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

// GetSabakanMachines returns all machines
func (g *gqlClient) GetSabakanMachines(ctx context.Context) ([]*machine, error) {
	greq := graphQLRequest{
		Query: `{
  searchMachines(having: null, notHaving: null) {
    spec {
      serial
      labels {
        name
        value
      }
      ipv4
      bmc {
        ipv4
      }
    }
    status {
      state
    }
  }
}`,
	}
	gdata, err := g.requestGQL(ctx, greq)
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
			BMCAddr:  m.Spec.BMC.IPv4,
			State:    toMachineState(m.Status.State),
		}
	}
	return ret, nil
}

// UpdateSabakanState updates given machine's state
func (g *gqlClient) UpdateSabakanState(ctx context.Context, serial string, state sabakan.MachineState) error {
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

	_, err := g.requestGQL(ctx, greq)
	return err
}
