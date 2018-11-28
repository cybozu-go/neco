package sabakan

import "fmt"
import "net/url"
import "strings"

// Query is an URL query
type Query map[string]string

// Match returns true if all non-empty fields matches Machine
func (q Query) Match(m *Machine) bool {
	if serial := q["serial"]; len(serial) > 0 && serial != m.Spec.Serial {
		return false
	}
	if ipv4 := q["ipv4"]; len(ipv4) > 0 {
		match := false
		for _, ip := range m.Spec.IPv4 {
			if ip == ipv4 {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	if ipv6 := q["ipv6"]; len(ipv6) > 0 {
		match := false
		for _, ip := range m.Spec.IPv6 {
			if ip == ipv6 {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	if labels := q["labels"]; len(labels) > 0 {
		// Split into each query
		rawQueries := strings.Split(labels, ",")
		for _, rawQuery := range rawQueries {
			rawQuery = strings.TrimSpace(rawQuery)
			query, err := url.ParseQuery(rawQuery)
			if err != nil {
				return false
			}
			for k, v := range query {
				if label, exists := m.Spec.Labels[k]; exists {
					if v[0] != label {
						return false
					}
				} else {
					return false
				}
			}
		}
	}
	if rack := q["rack"]; len(rack) > 0 && rack != fmt.Sprint(m.Spec.Rack) {
		return false
	}
	if role := q["role"]; len(role) > 0 && role != m.Spec.Role {
		return false
	}
	if bmc := q["bmc-type"]; len(bmc) > 0 && bmc != m.Spec.BMC.Type {
		return false
	}
	if state := q["state"]; len(state) > 0 && state != m.Status.State.String() {
		return false
	}

	return true
}

// Serial returns value of serial in the query
func (q Query) Serial() string { return q["serial"] }

// Rack returns value of rack in the query
func (q Query) Rack() string { return q["rack"] }

// Role returns value of role in the query
func (q Query) Role() string { return q["role"] }

// IPv4 returns value of ipv4 in the query
func (q Query) IPv4() string { return q["ipv4"] }

// IPv6 returns value of ipv6 in the query
func (q Query) IPv6() string { return q["ipv6"] }

// BMCType returns value of bmc-type in the query
func (q Query) BMCType() string { return q["bmc-type"] }

// State returns value of state the query
func (q Query) State() string { return q["state"] }

// Labels return label's key and value combined with '='
func (q Query) Labels() []string {
	queries := strings.Split(q["labels"], ",")
	for idx, rawQuery := range queries {
		queries[idx] = strings.TrimSpace(rawQuery)
	}
	return queries
}

// IsEmpty returns true if query is empty or no values are presented
func (q Query) IsEmpty() bool {
	for _, v := range q {
		if len(v) > 0 {
			return false
		}
	}
	return true
}
