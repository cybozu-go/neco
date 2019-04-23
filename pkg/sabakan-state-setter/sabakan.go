package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type graphQLRequest struct {
	Query string `json:"query"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
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

func requestGQL(ctx context.Context, client *http.Client, address string, data []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, address, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
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
		return nil, errors.New(gresp.Errors[0].Message)
	}
	return []byte(gresp.Data), nil
}

func getSabakanMachines(ctx context.Context, client *http.Client, address string) (*searchMachineResponse, error) {
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
	data, err := json.Marshal(greq)
	if err != nil {
		return nil, err
	}
	gdata, err := requestGQL(ctx, client, address, data)
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

func setSabakanStates(ctx context.Context, ms machineStateSource) error {
	return nil
}
