package gql

import (
	"time"

	"github.com/cybozu-go/sabakan/v2"
)

func matchMachine(m *sabakan.Machine, h, nh *MachineParams, now time.Time) bool {
	if !containsAllLabels(h, m.Spec.Labels) {
		return false
	}
	if containsAnyLabel(nh, m.Spec.Labels) {
		return false
	}

	if !containsRack(h, int(m.Spec.Rack), true) {
		return false
	}
	if containsRack(nh, int(m.Spec.Rack), false) {
		return false
	}

	if !containsRole(h, m.Spec.Role, true) {
		return false
	}
	if containsRole(nh, m.Spec.Role, false) {
		return false
	}

	if !containsState(h, m.Status.State, true) {
		return false
	}
	if containsState(nh, m.Status.State, false) {
		return false
	}

	days := int(m.Spec.RetireDate.Sub(now).Hours() / 24)
	if h != nil && h.MinDaysBeforeRetire != nil {
		if *h.MinDaysBeforeRetire > days {
			return false
		}
	}
	if nh != nil && nh.MinDaysBeforeRetire != nil {
		if *nh.MinDaysBeforeRetire <= days {
			return false
		}
	}

	return true
}

func containsAllLabels(h *MachineParams, labels map[string]string) bool {
	if h == nil {
		return true
	}
	for _, l := range h.Labels {
		v, ok := labels[l.Name]
		if !ok {
			return false
		}
		if v != l.Value {
			return false
		}
	}
	return true
}

func containsAnyLabel(h *MachineParams, labels map[string]string) bool {
	if h == nil {
		return false
	}
	for _, l := range h.Labels {
		v, ok := labels[l.Name]
		if !ok {
			continue
		}
		if v == l.Value {
			return true
		}
	}
	return false
}

func containsRack(h *MachineParams, target int, base bool) bool {
	if h == nil || len(h.Racks) == 0 {
		return base
	}
	for _, rack := range h.Racks {
		if rack == target {
			return true
		}
	}
	return false
}

func containsRole(h *MachineParams, target string, base bool) bool {
	if h == nil || len(h.Roles) == 0 {
		return base
	}
	for _, role := range h.Roles {
		if role == target {
			return true
		}
	}
	return false
}

func containsState(h *MachineParams, target sabakan.MachineState, base bool) bool {
	if h == nil || len(h.States) == 0 {
		return base
	}

	for _, state := range h.States {
		if state == target {
			return true
		}
	}
	return false
}
