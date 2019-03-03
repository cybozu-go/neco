package placemat

import "context"

func createNatRules() error {
	cmds := [][]string{}
	for _, iptables := range []string{"iptables", "ip6tables"} {
		cmds = append(cmds,
			[]string{iptables, "-N", "PLACEMAT", "-t", "filter"},
			[]string{iptables, "-N", "PLACEMAT", "-t", "nat"},

			[]string{iptables, "-t", "nat", "-A", "POSTROUTING", "-j", "PLACEMAT"},
			[]string{iptables, "-t", "filter", "-A", "FORWARD", "-j", "PLACEMAT"},
		)
	}

	return execCommands(context.Background(), cmds)
}

// destroyNetwork destroys a bridge and iptables rules by the name
func destroyNatRules() error {
	cmds := [][]string{}
	for _, iptables := range []string{"iptables", "ip6tables"} {
		cmds = append(cmds,
			[]string{iptables, "-t", "filter", "-D", "FORWARD", "-j", "PLACEMAT"},
			[]string{iptables, "-t", "nat", "-D", "POSTROUTING", "-j", "PLACEMAT"},

			[]string{iptables, "-F", "PLACEMAT", "-t", "filter"},
			[]string{iptables, "-X", "PLACEMAT", "-t", "filter"},

			[]string{iptables, "-F", "PLACEMAT", "-t", "nat"},
			[]string{iptables, "-X", "PLACEMAT", "-t", "nat"},
		)
	}
	return execCommandsForce(cmds)
}
