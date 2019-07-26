package sss

import serf "github.com/hashicorp/serf/client"

func getSerfMembers(client *serf.RPCClient) ([]serf.Member, error) {
	return client.Members()
}
