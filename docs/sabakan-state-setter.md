sabakan-state-setter
====================

`sabakan-state-setter` changes the states of machines. It has the following three functions.

1. Health check
    Decide sabakan machine states according to [serf][] statuses, [monitor-hw][] metrics, and [Prometheus][] alerts, and then update the states.
    Not all machines are checked by `sabakan-state-setter`. It observes `Uninitialized`, `Healthy`, `Unhealthy`, and `Unreachable` machines.
    Health check just updates sabakan machine states. There are no side effects.

2. Retirement
    `sabakan-state-setter` let retiring machines retire.
    When the `Retiring` machines exist, sabakan-state-setter will delete disk encryption keys on the sabakan.
    And clear TPM devices on the machines by `neco tpm clear`.
    If the retirement is succeeded, change the machine's state to `Retired`.
    The power state of retired machines after this retirement are depends on the TPM clear logic in `neco tpm clear`.

3. Shutdown
    Shutdown the `Retired` machines periodically.
    The execution cycle can be specified in a config file.

See machine state types at [Sabakan lifecycle management](https://github.com/cybozu-go/sabakan/blob/master/docs/lifecycle.md).

Health check
------------

### Strategy

`sabakan-state-setter` decides machine state by the strategy as follows:

- Judge as `Healthy`
  - serf status is `alive`.
  - serf tags `systemd-units-failed` is set and has no errors.
  - All of later mentioned machine peripherals are healthy.
  - There are no active Prometheus alerts that match any of `trigger-alerts` in the configuration file.
    - This condition includes the case when Prometheus Alertmanager is down.
- Judge as `Unreachable`
  - serf status is `failed`, `left` or machine is not yet as a serf member. It is the same as that `sabakan-state-setter` can not access monitor-hw metrics.
  - There is an active Prometheus alert which indicates that the machine is unreachable.
- Judge as `Unhealthy`
  - serf status is `alive`.
  - At least one of them matches:
    - serf tags `systemd-units-failed` is not set or has errors.
    - `sabakan-state-setter` can not retrieve monitor-hw metrics.
    - At least one of later mentioned machine peripherals is unhealthy.
  - There is an active Prometheus alert which indicates that the machine is unhealthy.
- Nothing to judge machine state
  - `sabakan-state-setter` can not access to `serf.service` of the same boot server.
  
### Grace period of setting unhealthy state

In order not to be too sensitive to temporary problem of machines' metrics,
sabakan-state-setter waits a grace period before updating a machine's state to `unhealthy`.

sabakan-state-setter updates the machine state
if and only if it judges the machine's state as `unhealthy` for the time specified in this value. 

Note that this grace period is not applied when an active Prometheus alert is the source of the state change.
We can configure Prometheus alerts to wait a sufficient amount of time before becoming active.

### Target machine peripherals

You can define the metrics used for health checking in in the configuration file.

The set of the metrics depends on its machine type. So you need configure a set of metrics for each machine type.

Basically, check the following peripherals.

- CPU
- Memory
- Storage controllers
- NVMe SSD
- [Dell BOSS][]
- Hard drives on the storage servers

Retirement
----------

`sabakan-state-setter` let retiring machines retire by the following steps:

1. Delete disk encryption keys on the sbakan.
2. Clear TPM devices on the machine by `neco tpm clear`.
3. Change the mahcine's state to `Retired`.

Shutdown
--------

`sabakan-state-setter` shutdown retired machines periodically.
The execution cycle can be specified in a config file.

Usage
-----

```console
sabakan-state-setter [OPTIONS]
```

| Option               | Default value             | Description                                                                       |
| -------------------- | ------------------------- | --------------------------------------------------------------------------------- |
| `-config-file`       | `''`                      | Path of config file.                                                              |
| `-etcd-session-ttl`  | `1m`                      | TTL of etcd session. This value is interpreted as a [duration string][].          |
| `-interval`          | `1m`                      | Interval of scraping metrics. This value is interpreted as a [duration string][]. |
| `-parallel`          | `30`                      | The number of parallel execution of getting machines metrics.                     |
| `-sabakan-url`       | `http://localhost:10080`  | sabakan HTTP Server URL.                                                          |
| `-sabakan-url-https` | `https://localhost:10443` | sabakan HTTPS Server URL.                                                         |
| `-serf-address`      | `127.0.0.1:7373`          | serf address.                                                                     |

Config file
-----------

| Field                                             | Default value | Description                                                                                                      |
| ------------------------------------------------- | ------------- | ---------------------------------------------------------------------------------------------------------------- |
| `shutdown-schedule` string                        | `""`          | Schedule in Cron format for retired machines shutdown. If this field is omitted, shutdown will not be performed. |
| `machine-types` [MachineType](#MachineType) array | `nil`         | Machine types is a list of `MachineType`. You should list all machine types used in your data center.            |
| `alert-monitor` \*[AlertMonitor](#AlertMonitor)   | `nil`         | Configurations to monitor Prometheus alerts.                                                                     |

### `MachineType`

| Field                             | Default value | Description                                                                                                 |
| --------------------------------- | ------------- | ----------------------------------------------------------------------------------------------------------- |
| `name` string                     |               | Name of this machine type. It is expected that this field is unique in setting file.                        |
| `metrics` [Metric](#Metric) array | `nil`         | Metrics is an array of `Metric` to be checked.                                                              |
| `grace-period` string             | `1h`          | Time to wait for updating machine state to `unhealthy`. This value is interpreted as a [duration string][]. |

### `Metric`

| Field                        | Default value | Description                                                                                                                                                                                                                                                    |
| ---------------------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name` string                | `''`          | Name of this metric.                                                                                                                                                                                                                                           |
| `selector` Selector          | nil           |                                                                                                                                                                                                                                                                |
| `minimum-healthy-count` *int | nil           | If the count of matching metrics whose value is not healthy is less than `minimum_healthy_count`, the machine is unhealthy.<br/>If `minimum_healthy_count` is `nil`, it means that if any one of the matching labels is not healthy, the machine is unhealthy. |

The meaning of `name` and `labels` are the same as Prometheus.
https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels

Please refer to the following link to know how to define `name` and `labels`.
https://github.com/cybozu-go/setup-hw/blob/master/docs/rule.md

### `Selector`

| Field                              | Default value | Description                                                     |
| ---------------------------------- | ------------- | --------------------------------------------------------------- |
| `labels` `map[string]string`       | `nil`         | Check all `name` metrics with labels matching exactly `labels`. |
| `label-prefix` `map[string]string` | `nil`         | Check all `name` metrics with labels having `labelPrefix`.      |

`labels` and `label-prefix` are AND condition,
i.e. a metric is selected if and only if all of the conditions are satisfied.

### `AlertMonitor`

| Field                                                | Default value | Description                                                                     |
| ---------------------------------------------------- | ------------- | ------------------------------------------------------------------------------- |
| `alertmanager-endpoint` string                       | `''`          | URL of Alertmanager API V2 endpoint (e.g., `http://alertmanager:9093/api/v2/`). |
| `trigger-alerts` [TriggerAlert](#TriggerAlert) array | `nil`         | Prometheus alerts that are recognized as indications of non-healthy machines.   |

### `TriggerAlert`

| Field                        | Default value | Description                                                                                                                                              |
| ---------------------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name` string                | `''`          | Name of a Prometheus alert to be monitored.                                                                                                              |
| `labels` `map[string]string` | `nil`         | Filtering labels and their values of a Prometheus alert. If a suspicious alert does not contain these labels, the alert is ignored.                      |
| `address-label` string       | `''`          | Label name of a Prometheus alert that denotes the IP address of a non-healthy machine. Exactly one of `address-label` and `serial-label` is required.    |
| `serial-label` string        | `''`          | Label name of a Prometheus alert that denotes the serial number of a non-healthy machine. Exactly one of `address-label` and `serial-label` is required. |
| `state` string               | `''`          | Candidate of the next state of a non-healthy machine. Currently `unreachable` and `unhealthy` are supported.                                             |

[Dell BOSS]: https://i.dell.com/sites/doccontent/shared-content/data-sheets/en/Documents/Dell-PowerEdge-Boot-Optimized-Storage-Solution.pdf
[duration string]: https://golang.org/pkg/time/#ParseDuration
[monitor-hw]: https://github.com/cybozu-go/setup-hw/blob/master/docs/monitor-hw.md
[serf]: https://www.serf.io/
[Prometheus]: https://prometheus.io/
