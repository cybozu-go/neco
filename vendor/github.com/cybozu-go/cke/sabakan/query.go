package sabakan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
)

var httpClient = &well.HTTPClient{
	Client: &http.Client{},
}

// GraphQLQuery is GraphQL query to retrieve machine information from sabakan.
const GraphQLQuery = `
query ckeSearch($having: MachineParams = null,
                $notHaving: MachineParams = {
                  roles: ["boot"]
                  states: [RETIRED]
                }) {
  searchMachines(having: $having, notHaving: $notHaving) {
    spec {
      serial
      labels {
        name
        value
      }
      rack
      indexInRack
      role
      ipv4
      registerDate
      retireDate
    }
    status {
      state
      duration
    }
  }
}
`

// State is the enum type for sabakan states.
type State string

// SabakanState list defined in GraphQL schema.
const (
	StateUninitialized = State("UNINITIALIZED")
	StateHealthy       = State("HEALTHY")
	StateUnhealthy     = State("UNHEALTHY")
	StateUnreachable   = State("UNREACHABLE")
	StateUpdating      = State("UPDATING")
	StateRetiring      = State("RETIRING")
	StateRetired       = State("RETIRED")
)

// IsValid returns true if s is vaild.
func (s State) IsValid() bool {
	switch s {
	case StateUninitialized:
	case StateHealthy:
	case StateUnhealthy:
	case StateUnreachable:
	case StateUpdating:
	case StateRetiring:
	case StateRetired:

	default:
		return false
	}
	return true
}

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

// IsValid returns non-nil error if mp is not valid.
func (mp MachineParams) IsValid() error {
	for _, state := range mp.States {
		if !state.IsValid() {
			return errors.New("invalid state: " + string(state))
		}
	}

	return nil
}

// QueryVariables represents the JSON object of the query variables.
type QueryVariables struct {
	Having    *MachineParams `json:"having"`
	NotHaving *MachineParams `json:"notHaving"`
}

// IsValid returns non-nil error if v is not valid.
func (v QueryVariables) IsValid() error {
	if v.Having != nil {
		if err := v.Having.IsValid(); err != nil {
			return err
		}
	}
	if v.NotHaving != nil {
		if err := v.NotHaving.IsValid(); err != nil {
			return err
		}
	}
	return nil
}

// Machine represents a machine registered with sabakan.
type Machine struct {
	Spec struct {
		Serial string `json:"serial"`
		Labels []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"labels"`
		Rack         int       `json:"rack"`
		IndexInRack  int       `json:"indexInRack"`
		Role         string    `json:"role"`
		IPv4         []string  `json:"ipv4"`
		RegisterDate time.Time `json:"registerDate"`
		RetireDate   time.Time `json:"retireDate"`
	} `json:"spec"`
	Status struct {
		State    State   `json:"state"`
		Duration float64 `json:"duration"`
	} `json:"status"`
}

// Query send a GraphQL query to sabakan.
// If sabakan URL is not set, this returns (nil, cke.ErrNotFound).
func Query(ctx context.Context, storage cke.Storage) ([]Machine, error) {
	url, err := storage.GetSabakanURL(ctx)
	if err != nil {
		return nil, err
	}

	var variables *QueryVariables
	varsData, err := storage.GetSabakanQueryVariables(ctx)
	switch err {
	case cke.ErrNotFound:
	case nil:
		variables = new(QueryVariables)
		err = json.Unmarshal(varsData, variables)
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	return doQuery(ctx, url, variables, httpClient)
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
