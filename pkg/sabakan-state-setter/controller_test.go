package sss

import (
	"context"
	"errors"
	"testing"
	"time"

	sabakan "github.com/cybozu-go/sabakan/v3"
)

func newMockController(saba *sabakanMockClient, prom *promMockClient, serf *serfMockClient, alertmanager *alertmanagerClient, neco *necoCmdMockExecutor, mt ...*machineType) *Controller {
	machineTypes := map[string]*machineType{}
	for _, m := range mt {
		machineTypes[m.Name] = m
	}

	return &Controller{
		interval:           time.Minute,
		parallelSize:       2,
		sabakanClient:      saba,
		promClient:         prom,
		serfClient:         serf,
		alertmanagerClient: alertmanager,
		necoExecutor:       neco,
		machineTypes:       machineTypes,
		unhealthyMachines:  make(map[string]time.Time),
	}
}

var machineTypeSerfOnly = &machineType{
	Name: "serfonly",
	GracePeriod: duration{
		Duration: time.Millisecond,
	},
	MetricsCheckList: []targetMetric{},
}

var triggerAlerts = []triggerAlert{
	{
		Name:        "DiskNotRecognized",
		SerialLabel: "serial",
		State:       sabakan.StateUnhealthy,
		Labels: map[string]string{
			"severity": "error",
		},
	},
	{
		Name:         "LLDPDown",
		AddressLabel: "address",
		State:        sabakan.StateUnreachable,
	},
}

func testControllerRun(t *testing.T) {
	t.Parallel()

	machineTypeQEMU := &machineType{
		Name: "qemu",
		GracePeriod: duration{
			Duration: time.Millisecond,
		},
		MetricsCheckList: []targetMetric{
			{
				Name: "hw_processor_status_health",
			},
			{
				Name: "hw_storage_controller_status_health",
				Selector: &selector{
					Labels: map[string]string{
						"controller": "PCIeSSD.Slot.2-C",
						"system":     "System.Embedded.1",
					},
				},
			},
			{
				Name: "hw_storage_controller_status_health",
				Selector: &selector{
					Labels: map[string]string{"controller": "PCIeSSD.Slot.3-C"},
				},
			},
			{
				Name: "hw_storage_controller_status_health",
				Selector: &selector{
					LabelPrefix: map[string]string{
						"controller": "SATAHDD.Slot.",
						"system":     "System.Embedded.",
					},
				},
				MinimumHealthyCount: intPointer(1),
			},
		},
	}
	machineTypeLongWait := &machineType{
		Name: "longwait",
		GracePeriod: duration{
			Duration: time.Hour,
		},
	}

	testCases := []struct {
		message    string
		machines   []*machine
		serfStatus map[string]*serfStatus
		metrics    map[string]string
		alerts     []map[string]string

		expected map[string]sabakan.MachineState
	}{
		{
			message: "do health check for some machines",
			machines: []*machine{
				{
					Serial:   "00000000",
					Type:     "qemu",
					IPv4Addr: "10.0.0.100",
					State:    sabakan.StateUninitialized,
				},
				{
					Serial:   "00000001",
					Type:     "qemu",
					IPv4Addr: "10.0.0.101",
					State:    sabakan.StateUninitialized,
				},
				{
					Serial:   "00000002",
					Type:     "qemu",
					IPv4Addr: "10.0.0.102",
					State:    sabakan.StateUninitialized,
				},
				{
					Serial:   "00000003",
					Type:     "qemu",
					IPv4Addr: "10.0.0.103",
					State:    sabakan.StateHealthy,
				},
				{
					Serial:   "00000004",
					Type:     "longwait",
					IPv4Addr: "10.0.0.104",
					State:    sabakan.StateHealthy,
				},
				{
					Serial:   "00000005",
					Type:     "longwait",
					IPv4Addr: "10.0.0.105",
					State:    sabakan.StateHealthy,
				},
			},
			serfStatus: map[string]*serfStatus{
				"10.0.0.100": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.101": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.102": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.103": {
					Status:             "failed",
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.104": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.105": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
			},
			metrics: map[string]string{
				"10.0.0.100": `
# TYPE hw_processor_status_health gauge
hw_processor_status_health{processor="CPU.Socket.1"} 0
hw_processor_status_health{processor="CPU.Socket.2"} 1
`,
				"10.0.0.101": `
# TYPE hw_processor_status_health gauge
hw_processor_status_health{processor="CPU.Socket.1"} 0
hw_processor_status_health{processor="CPU.Socket.2"} 0
# TYPE hw_storage_controller_status_health gauge
hw_storage_controller_status_health{controller="SATAHDD.Slot.1"} 1
hw_storage_controller_status_health{controller="SATAHDD.Slot.2"} 1
`,
				"10.0.0.102": `
# TYPE hw_processor_status_health gauge
hw_processor_status_health{processor="CPU.Socket.1"} 0
hw_processor_status_health{processor="CPU.Socket.2"} 0
# TYPE hw_storage_controller_status_health gauge
hw_storage_controller_status_health{controller="PCIeSSD.Slot.2-C", system="System.Embedded.1"} 0
hw_storage_controller_status_health{controller="PCIeSSD.Slot.3-C", system="System.Embedded.1"} 0
hw_storage_controller_status_health{controller="SATAHDD.Slot.1", system="System.Embedded.1"} 0
hw_storage_controller_status_health{controller="SATAHDD.Slot.2", system="System.Embedded.1"} 1
`,
			},
			alerts: []map[string]string{
				{
					"alertname": "DiskNotRecognized",
					"serial":    "00000002",
					"severity":  "warning",
				},
				{
					"alertname": "DiskNotRecognized",
					"serial":    "00000004",
					"severity":  "error",
				},
				{
					"alertname": "LLDPDown",
					"address":   "10.0.0.105",
				},
				{
					"alertname": "DiskNotRecognized",
					"serial":    "00000005",
					"severity":  "error",
				},
			},
			expected: map[string]sabakan.MachineState{
				"00000000": sabakan.StateUnhealthy,   // one of two CPUs is issuing a warning
				"00000001": sabakan.StateUnhealthy,   // all HDD are unhealthy; # of healthy HDDs falls below MinimumHealthyCount (0 < 1)
				"00000002": sabakan.StateHealthy,     // one of two HDDs is unhealthy, but it is acceptable; alert "DiskNotRecognized" is firing but it is still warning
				"00000003": sabakan.StateUnreachable, // serf status is "failed"
				"00000004": sabakan.StateUnhealthy,   // alert "DiskNotRecognized" is firing; grace period is ignored
				"00000005": sabakan.StateUnreachable, // alert "LLDPDown" is firing; alert "DiskNotRecognized" is ignored because it is less severe
			},
		},
		{
			message: "skip health check or retirement",
			machines: []*machine{
				{
					Serial:   "uninitialized",
					Type:     "serfonly",
					IPv4Addr: "10.0.0.100",
					State:    sabakan.StateUninitialized,
				},
				{
					Serial:   "healthy",
					Type:     "serfonly",
					IPv4Addr: "10.0.0.101",
					State:    sabakan.StateHealthy,
				},
				{
					Serial:   "unhealthy",
					Type:     "serfonly",
					IPv4Addr: "10.0.0.102",
					State:    sabakan.StateUnhealthy,
				},
				{
					Serial:   "unreachable",
					Type:     "serfonly",
					IPv4Addr: "10.0.0.103",
					State:    sabakan.StateUnreachable,
				},
				{
					Serial:   "updating",
					Type:     "serfonly",
					IPv4Addr: "10.0.0.104",
					State:    sabakan.StateUpdating,
				},
				{
					Serial:   "retiring",
					Type:     "serfonly",
					IPv4Addr: "10.0.0.105",
					State:    sabakan.StateRetiring,
				},
				{
					Serial:   "retired",
					Type:     "serfonly",
					IPv4Addr: "10.0.0.106",
					State:    sabakan.StateRetired,
				},
			},
			serfStatus: map[string]*serfStatus{
				"10.0.0.100": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.101": {
					Status:             "failed", // Healthy -> Unreachable
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.102": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.103": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.104": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.105": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
				"10.0.0.106": {
					Status:             "alive",
					SystemdUnitsFailed: strPtr(""),
				},
			},
			expected: map[string]sabakan.MachineState{
				"uninitialized": sabakan.StateHealthy,     // healthcheck
				"healthy":       sabakan.StateUnreachable, // healthcheck
				"unhealthy":     sabakan.StateHealthy,     // healthcheck
				"unreachable":   sabakan.StateHealthy,     // healthcheck
				"updating":      sabakan.StateUpdating,    // nothing to do
				"retiring":      sabakan.StateRetired,     // retirement
				"retired":       sabakan.StateRetired,     // nothing to do
			},
		},
	}

	for _, tc := range testCases {
		sabaMock := newMockSabakanClient(tc.machines)
		promMock := newMockPromClient(tc.metrics)
		serfMock, _ := newMockSerfClient(tc.serfStatus)
		alertmanagerMock := newMockAlertmanagerClient(triggerAlerts, tc.alerts, nil)
		necoMock := newMockNecoCmdExecutor()
		ctr := newMockController(sabaMock, promMock, serfMock, alertmanagerMock, necoMock, machineTypeQEMU, machineTypeSerfOnly, machineTypeLongWait)
		for i := 0; i < 2; i++ {
			err := ctr.runOnce(context.Background())
			if err != nil {
				t.Error(err)
			}
			time.Sleep(100 * time.Millisecond)
		}
		for serial, expectedState := range tc.expected {
			if sabaMock.getState(serial) != expectedState {
				t.Error(tc.message, "serial:", serial, "expected:", expectedState, "actual:", sabaMock.getState(serial))
			}
		}
	}
}

func testControllerRunSerfError(t *testing.T) {
	t.Parallel()

	// When failed to get serf status, controllr should return an error.
	machines := []*machine{
		{
			Serial:   "00000000",
			Type:     "serfonly",
			IPv4Addr: "10.0.0.100",
			State:    sabakan.StateUninitialized,
		},
	}

	sabaMock := newMockSabakanClient(machines)
	promMock := newMockPromClient(map[string]string{})
	serfMock, _ := newMockSerfClient(nil) // serfMockClient will return an error.
	necoMock := newMockNecoCmdExecutor()
	ctr := newMockController(sabaMock, promMock, serfMock, nil, necoMock, machineTypeSerfOnly)
	err := ctr.runOnce(context.Background())
	if err == nil {
		t.Error("return nil")
	}
}

func testControllerRunAlertmanagerError(t *testing.T) {
	t.Parallel()

	machines := []*machine{
		{
			Serial:   "00000000",
			Type:     "serfonly",
			IPv4Addr: "10.0.0.100",
			State:    sabakan.StateUninitialized,
		},
	}

	sabaMock := newMockSabakanClient(machines)
	promMock := newMockPromClient(map[string]string{})
	serfMock, _ := newMockSerfClient(map[string]*serfStatus{
		"10.0.0.100": {
			Status:             "alive",
			SystemdUnitsFailed: strPtr(""),
		},
	})
	alertmanagerMock := newMockAlertmanagerClient(triggerAlerts, nil, errors.New("failed to connect to Alertmanager")) // alertmanagerMock will return an error. The error will be ignored.
	necoMock := newMockNecoCmdExecutor()
	ctr := newMockController(sabaMock, promMock, serfMock, alertmanagerMock, necoMock, machineTypeSerfOnly)

	err := ctr.runOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if sabaMock.getState("00000000") != sabakan.StateHealthy {
		t.Error(`sabaMock.getState("00000000") != sabakan.StateHealthy`)
	}
}

func testControllerUnhealthy(t *testing.T) {
	t.Parallel()

	mt := &machineType{
		Name: "type1",
		GracePeriod: duration{
			Duration: time.Minute * 60,
		},
	}
	m1 := &machine{
		Serial: "1",
		Type:   "type1",
	}
	m2 := &machine{
		Serial: "2",
		Type:   "type1",
	}
	baseTime := time.Now()

	ctr := newMockController(nil, nil, nil, nil, nil, mt)

	exceeded := ctr.RegisterUnhealthy(m1, baseTime)
	if exceeded {
		t.Error("machine is misjudged as long-term unhealthy at the first registration")
	}

	exceeded = ctr.RegisterUnhealthy(m1, baseTime.Add(time.Minute*30))
	if exceeded {
		t.Error("machine is misjudged as long-term unhealthy during grace period")
	}

	exceeded = ctr.RegisterUnhealthy(m1, baseTime.Add(time.Minute*70)) // 60 < 70 < 30+60
	if !exceeded {
		t.Error("machine is not judged as long-term unhealthy after grace period")
	}

	ctr.ClearUnhealthy(m1)

	exceeded = ctr.RegisterUnhealthy(m1, baseTime.Add(time.Minute*80))
	if exceeded {
		t.Error("machine is misjudged as long-term unhealthy after clearing registry")
	}

	exceeded = ctr.RegisterUnhealthy(m2, baseTime.Add(time.Minute*150)) // 150 > 80+60
	if exceeded {
		t.Error("machine is misjudged as long-term unhealthy by confusion")
	}
}

func testControllerRetire(t *testing.T) {
	t.Parallel()

	machines := []*machine{
		{
			Serial:   "retiring0",
			Type:     "serfonly",
			IPv4Addr: "10.0.0.100",
			State:    sabakan.StateRetiring,
		},
		{
			Serial:   "retiring1",
			Type:     "serfonly",
			IPv4Addr: "10.0.0.101",
			State:    sabakan.StateRetiring,
		},
		{
			Serial:   "retired",
			Type:     "serfonly",
			IPv4Addr: "10.0.0.102",
			State:    sabakan.StateRetired,
		},
		{
			Serial:   "healthy",
			Type:     "serfonly",
			IPv4Addr: "10.0.0.103",
			State:    sabakan.StateHealthy,
		},
	}

	sabaMock := newMockSabakanClient(machines)
	promMock := newMockPromClient(map[string]string{})
	serfMock, _ := newMockSerfClient(map[string]*serfStatus{})
	necoMock := newMockNecoCmdExecutor()
	ctr := newMockController(sabaMock, promMock, serfMock, nil, necoMock, machineTypeSerfOnly)

	stateMap := ctr.machineRetire(context.Background(), machines)
	if stateMap == nil {
		t.Error("return nil")
	}

	// confirm that the "retiring" machines will be "retired".
	for _, serial := range []string{"retiring0", "retiring1"} {
		if stateMap[serial] != sabakan.StateRetired {
			t.Error(serial, "machine state is not Retired")
		}
		if sabaMock.getCryptsDeleteCount(serial) != 1 {
			t.Error(serial, "sabakan.CryptsDelete is not called")
		}
		if necoMock.getTPMClearCount(serial) != 1 {
			t.Error(serial, "'neco tpm clear' is not called")
		}
	}

	// confirm that other machines will NOT be changed.
	for _, serial := range []string{"retired", "healthy"} {
		newState, ok := stateMap[serial]
		if ok {
			t.Error(serial, "machine state will be changed: "+newState)
		}
		if sabaMock.getCryptsDeleteCount(serial) != 0 {
			t.Error(serial, "sabakan.CryptsDelete is called")
		}
		if necoMock.getTPMClearCount(serial) != 0 {
			t.Error(serial, "'neco tpm clear' is called")
		}
	}
}

func testControllerShutdown(t *testing.T) {
	t.Parallel()

	machines := []*machine{
		{
			Serial:   "retired-power-off",
			Type:     "serfonly",
			IPv4Addr: "10.0.0.100",
			State:    sabakan.StateRetired,
		},
		{
			Serial:   "retired-power-on",
			Type:     "serfonly",
			IPv4Addr: "10.0.0.101",
			State:    sabakan.StateRetired,
		},
		{
			Serial:   "healthy",
			Type:     "serfonly",
			IPv4Addr: "10.0.0.103",
			State:    sabakan.StateHealthy,
		},
		{
			Serial:   "retiring",
			Type:     "serfonly",
			IPv4Addr: "10.0.0.104",
			State:    sabakan.StateRetiring,
		},
	}

	// set initial power state for retired machines.
	necoMock := newMockNecoCmdExecutor()
	necoMock.setPowerState("retired-power-off", "Off")
	necoMock.setPowerState("retired-power-on", "On")

	sabaMock := newMockSabakanClient(machines)
	promMock := newMockPromClient(map[string]string{})
	serfMock, _ := newMockSerfClient(map[string]*serfStatus{})
	ctr := newMockController(sabaMock, promMock, serfMock, nil, necoMock, machineTypeSerfOnly)

	ctr.machineShutdown(context.Background())

	// confirm that the "retired-power-off" machines will NOT be powered off.
	if necoMock.getPowerStopCount("retired-power-off") != 0 {
		t.Error("retired-power-off", "'neco power stop' is called")
	}

	// confirm that the "retired-power-on" machines will be powered off.
	if necoMock.getPowerStopCount("retired-power-on") == 0 {
		t.Error("retired-power-on", "'neco power stop' is not called")
	}

	// confirm that other machines will NOT be changed.
	for _, serial := range []string{"healthy", "retiring"} {
		if necoMock.getPowerStopCount(serial) != 0 {
			t.Error(serial, "'neco power stop' is called")
		}
		if necoMock.getPowerStatusCount(serial) != 0 {
			t.Error(serial, "'neco power status' is called")
		}
	}
}

func TestController(t *testing.T) {
	t.Run("Run", testControllerRun)
	t.Run("RunSerfError", testControllerRunSerfError)
	t.Run("RunAlertmanagerError", testControllerRunAlertmanagerError)
	t.Run("Unhealthy", testControllerUnhealthy)
	t.Run("Retire", testControllerRetire)
	t.Run("Shutdown", testControllerShutdown)
}
