package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/cybozu-go/neco/ext"
	"github.com/vektah/gqlparser/gqlerror"
)

type graphQLRequest struct {
	Query string `json:"query"`
}

type graphQLResponse struct {
	Data   json.RawMessage  `json:"data"`
	Errors []gqlerror.Error `json:"errors,omitempty"`
}

type searchMachineResponse struct {
	SearchMachines []machine `json:"searchMachines"`
}

type machine struct {
	Spec spec `json:"spec"`
}

type spec struct {
	Serial string   `json:"serial"`
	IPv4   []string `json:"ipv4"`
}

type gqlClient struct {
	httpClient *http.Client
	endpoint   string
}

func newGQLClient(address string) (*gqlClient, error) {
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

func (g *gqlClient) getSabakanMachines(ctx context.Context) (*searchMachineResponse, error) {
	greq := graphQLRequest{
		Query: `{
  searchMachines(having: null, notHaving: null) {
    spec {
      serial
      ipv4
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

	return resp, nil
}

func (g *gqlClient) setSabakanState(ctx context.Context, ms machineStateSource, state string) error {
	greq := graphQLRequest{
		Query: fmt.Sprintf(`mutation {
  setMachineState(serial: "%s", state: %s) {
    state
  }
}`, ms.serial, state),
	}

	_, err := g.requestGQL(ctx, greq)
	return err
}
