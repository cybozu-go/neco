package sss

import (
	"net/url"

	"github.com/cybozu-go/sabakan/v3"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/common/model"
)

type alertStatus struct {
	AlertName string
	State     sabakan.MachineState
}

type alertmanagerClient struct {
	client        alert.ClientService
	triggerAlerts []triggerAlert
}

func newAlertmanagerClient(alertMonitor *alertMonitor) (*alertmanagerClient, error) {
	if alertMonitor == nil {
		return nil, nil
	}

	u, err := url.Parse(alertMonitor.AlertmanagerEndpoint)
	if err != nil {
		return nil, err
	}

	config := &client.TransportConfig{
		Host:     u.Host,
		BasePath: u.Path,
		Schemes:  []string{u.Scheme},
	}
	client := &alertmanagerClient{
		client:        client.NewHTTPClientWithConfig(nil, config).Alert,
		triggerAlerts: alertMonitor.TriggerAlerts,
	}
	return client, nil
}

// GetAlertStatuses returns IP addresses and new states of non-healthy machines.
//
// `GetAlertStatuses` first retrieves non-silenced alerts from Alertmanager by using `ac.client`.
// `GetAlertStatuses` then scans the alerts to find non-healthy machines.
// `GetAlertStatuses` considers a machine non-healthy if some alert is active about that machine and the alert matches one of `ac.triggerAlerts`.
// `GetAlertStatuses` returns a mapping from non-healthy machines' IP addresses to `alertStatus`es, each of which describes the alert name and the new state.
func (ac *alertmanagerClient) GetAlertStatuses(machines []*machine) (map[string]*alertStatus, error) {
	if ac == nil {
		return nil, nil
	}

	// retrieve non-silenced alerts
	f := false
	params := alert.NewGetAlertsParams().WithSilenced(&f)
	rawAlerts, err := ac.client.GetAlerts(params)
	if err != nil {
		return nil, err
	}

	// simplify observed alerts; only the labels are important
	alerts := make(map[string][]map[string]string)
	for _, rawAlert := range rawAlerts.Payload {
		name, ok := rawAlert.Labels[model.AlertNameLabel]
		if !ok {
			continue
		}

		alerts[name] = append(alerts[name], rawAlert.Labels)
	}

	machinesByAddress := make(map[string]*machine)
	machinesBySerial := make(map[string]*machine)
	for _, machine := range machines {
		machinesByAddress[machine.IPv4Addr] = machine
		machinesBySerial[machine.Serial] = machine
	}

	statuses := make(map[string]*alertStatus)

	// scan observed alerts to check matching with trigger alerts
	for _, triggerAlert := range ac.triggerAlerts {
		for _, alert := range alerts[triggerAlert.Name] {
			if !isSubsetAlertLabels(triggerAlert.Labels, alert) {
				continue
			}

			var target *machine
			if triggerAlert.AddressLabel != "" {
				address, ok := alert[triggerAlert.AddressLabel]
				if !ok {
					continue
				}
				target, ok = machinesByAddress[address]
				if !ok {
					continue
				}
			}
			if triggerAlert.SerialLabel != "" {
				serial, ok := alert[triggerAlert.SerialLabel]
				if !ok {
					continue
				}
				target, ok = machinesBySerial[serial]
				if !ok {
					continue
				}
			}
			if target == nil {
				continue
			}

			if statuses[target.IPv4Addr] != nil && statuses[target.IPv4Addr].State == sabakan.StateUnreachable {
				// do not overwrite with less severe state
				continue
			}
			statuses[target.IPv4Addr] = &alertStatus{
				AlertName: triggerAlert.Name,
				State:     triggerAlert.State,
			}
		}
	}

	return statuses, nil
}

func isSubsetAlertLabels(triggerLabels map[string]string, actualLabels map[string]string) bool {
	for k, v := range triggerLabels {
		actual, ok := actualLabels[k]
		if !ok || actual != v {
			return false
		}
	}
	return true
}
