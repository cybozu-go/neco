package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	UpdateStatePrefix = "/api/v1/state/"
	UpdateLabelPrefix = "/api/v1/labels/"

	StateHealthy       = "healthy"
	StateUninitialized = "uninitialized"
	StateUnreachable   = "unreachable"
)

type sabakan struct {
	endpoint string
}

func (c sabakan) post(url string, body io.Reader) error {
	resp, err := http.Post(url, "", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if code := resp.StatusCode; code < 200 || code >= 400 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("sabakan returned %d: %s", code, body)
	}
	return nil
}
func (c sabakan) updateState(serial string, state string) error {
	url := c.endpoint + UpdateStatePrefix + serial
	return c.post(url, strings.NewReader(state))
}

func (c sabakan) updateOSVersion(serial string, version string) error {
	url := c.endpoint + UpdateLabelPrefix + serial
	data := map[string]string{"os-version": version}
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(data)
	if err != nil {
		return err
	}
	return c.post(url, body)
}
