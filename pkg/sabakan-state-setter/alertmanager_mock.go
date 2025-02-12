package sss

import (
	"github.com/go-openapi/runtime"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
)

// alertmanagerMockClient is a mock of alert.ClientService.
//
// Note that this is not a mock of alertmanagerClient.
type alertmanagerMockClient struct {
	alerts []map[string]string
	err    error
}

var _ alert.ClientService = &alertmanagerMockClient{}

// newMockAlertmanagerClient returns an instance of alertmanagerClient using a mock of alert.ClientService.
func newMockAlertmanagerClient(triggerAlerts []triggerAlert, alerts []map[string]string, err error) *alertmanagerClient {
	return &alertmanagerClient{
		client: &alertmanagerMockClient{
			alerts: alerts,
			err:    err,
		},
		triggerAlerts: triggerAlerts,
	}
}

func (ac *alertmanagerMockClient) GetAlerts(params *alert.GetAlertsParams, opts ...alert.ClientOption) (*alert.GetAlertsOK, error) {
	if ac.err != nil {
		return nil, ac.err
	}

	var gettableAlerts models.GettableAlerts
	for _, alert := range ac.alerts {
		gettableAlerts = append(gettableAlerts, &models.GettableAlert{
			Alert: models.Alert{
				Labels: alert,
			},
		})
	}

	return &alert.GetAlertsOK{
		Payload: gettableAlerts,
	}, nil
}

func (ac *alertmanagerMockClient) PostAlerts(params *alert.PostAlertsParams, opts ...alert.ClientOption) (*alert.PostAlertsOK, error) {
	return nil, nil
}

func (ac *alertmanagerMockClient) SetTransport(transport runtime.ClientTransport) {
}
