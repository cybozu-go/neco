package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCluster(t *testing.T) {
	data, err := os.ReadFile("../cluster.json.example")
	if err != nil {
		t.Fatal(err)
	}

	var clusters []Cluster
	err = json.Unmarshal(data, &clusters)
	if err != nil {
		t.Fatal(err)
	}

	if len(clusters) != 1 {
		t.Fatal(`len(clusters) != 1`)
	}

	c := clusters[0]
	if err := c.Validate(); err != nil {
		t.Error(err)
	}
}
