sabakan-state-setter
====================

This command set sabakan machine states according to [serf][] status and [monitor-hw][] metrics.
It runs by systemd oneshot service which is executed periodically by `systemd.timer`.

Machine state selection
-----------------------

See machine state types at [Sabakan lifecycle management](https://github.com/cybozu-go/sabakan/blob/master/docs/lifecycle.md).

`sabakan-state-setter` decides machine state by the strategy as follows:

- Judge as `Healthy`
  - serf status is `alive`.
  - serf tags `systemd-units-failed` is set and has no errors.
  - All of later mentioned machine peripherals are healthy.
- Judge as `Unreachable`
  - serf status is `failed`, `left` or machine is not yet as a serf member. It is the same as that `sabakan-state-setter` can not access monitor-hw metrics.
- Judge as `Unhealthy`
  - serf status is `alive`.
  - At least one of them matches:
    - serf tags `systemd-units-failed` is not set or has errors.
    - `sabakan-state-setter` can not retrieve monitor-hw metrics.
    - At least one of later mentioned machine peripherals is unhealthy.
- Nothing to judge machine state
  - `sabakan-state-setter` can not access to `serf.service` of the same boot server.
  
Grace period of setting problematic state
-----------------------------------------

In order not to be too sensitive to temporary unavailable of machines' metrics,
sabakan-state-setter waits a grace period before updating a machine's state to `(unhealthy|unreachable)`.

sabakan-state-setter updates the machine state
if and only if it judges the machine's state as `(unhealthy|unreachable)` for the time specified in this value. 

To save when the machine has been `(unhealthy|unreachable)`, sabakan-state-setter writes JSON on local path.
The path can be configured by option `--problematic-machines-file`.

The JSON consists of `problematic-machine` array.
`problematic-machine` is a machine whose state is `(unhealthy|unreachable)`.
The others are not listed in this JSON.

```json
[
   { "name": "rack0-cs1", "address":  "10.69.0.4", "state":  "unhealthy", "first_detection":  "2019-07-10 23:00:00 +0000 UTC"},
   { "name": "rack0-cs3", "address":  "10.69.0.6", "state":  "unreachable", "first_detection":  "2019-07-10 23:01:02 +0000 UTC"},
   ...
]
```

### `problematic-machine`

The format of `problematic-machine` is following:

| Key                 | Type                     | Description                                         |
| ------------------- | ------------------------ | --------------                                      |
| `name`              | string                   | Server's name                                       |
| `address`           | string                   | Server's address                                    |
| `state`             | string                   | Server's state(`unhealthy` or `unreachable`)        |
| `first_detection`   | string (UTC timestamp)   | UTC timestamp of the server was judged as the state |

Target machine peripherals
--------------------------

- CPU
- Memory
- Storage controllers
- NVMe SSD
- [Dell BOSS][]
- **Planned:** Hard drive on the storage servers

Describe in the configuration file the metrics names with labels.

Usage
-----

```console
sabakan-state-setter [OPTIONS]
```

| Option                        | Default value                                         | Description                                               |
| -------------------           | ------------------------                              | --------------                                            |
| `--sabakan-address`           | `http://localhost:10080`                              | URL of sabakan                                            |
| `--config-file`               | `''`                                                  | Path of config file                                       |
| `--problematic-machines-file` | `/run/sabakan-state-setter/problematic-machines.json` | Path of the machines whose state is problematic list JSON |

Problematic state means:
- `unhealthy`
- `unreachable`


Settings of target machine peripherals
--------------------------------------

The set of metrics used for health checking depends on its machine type.
You can configure a set of metrics to scrape for each machine type.

| Field                                             | Default value            | Description                                                                                           |
| -------------------                               | ------------------------ | --------------                                                                                        |
| `machine-types` [MachineType](#MachineType) array | `nil`                    | Machine types is a list of `MachineType`. You should list all machine types used in your data center. |

If all metrics defined in the config file are healthy, the machine is healthy. Otherwise, it's unhealthy.

### `MachineType`
| Field                                              | Default value            | Description                                                                                                                                               |
| -------------------                                | ------------------------ | --------------                                                                                                                                            |
| `name` string                                      | `''`                     | Name of this machine type. It is expected that this field is unique in setting file.                                                                      |
| `metrics` [Metric](#Metric) array                  | `nil`                    | Metrics is an array of `Metric` to be checked.                                                                                                            |
| `grace-period-of-setting-problematic-state` string | `'1m'`                   | Time to wait for updating machine state to problematic one. This value is interpreted as a [duration string](https://golang.org/pkg/time/#ParseDuration). |

### `Metric`

| Field                        | Default value            | Description                                                                                                                                                                                                                                                    |
| -------------------          | ------------------------ | --------------                                                                                                                                                                                                                                                 |
| `name` string                | `''`                     | Name of this metric.                                                                                                                                                                                                                                           |
| `selector` Selector          | nil                      |                                                                                                                                                                                                                                                                |
| `minimum-healthy-count` *int | nil                      | If the count of matching metrics whose value is not healthy is less than `minimum_healthy_count`, the machine is unhealthy.<br/>If `minimum_healthy_count` is `nil`, it means that if any one of the matching labels is not healthy, the machine is unhealthy. |

The meaning of `name` and `labels` are the same as Prometheus.
https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels

Please refer to the following link to know how to define `name` and `labels`.
https://github.com/cybozu-go/setup-hw/blob/master/docs/rule.md

### `Selector`

| Field                              | Default value            | Description                                                     |
| -------------------                | ------------------------ | --------------                                                  |
| `labels` `map[string]string`       | `nil`                    | Check all `name` metrics with labels matching exactly `labels`. |
| `label-prefix` `map[string]string` | `nil`                    | Check all `name` metrics with labels having `labelPrefix`.      |

`labels` and `label-prefix` are AND condition,
i.e. a metric is selected if and only if all of the conditions are satisfied.


[serf]: https://www.serf.io/
[monitor-hw]: https://github.com/cybozu-go/setup-hw/blob/master/docs/monitor-hw.md
[Dell BOSS]: https://i.dell.com/sites/doccontent/shared-content/data-sheets/en/Documents/Dell-PowerEdge-Boot-Optimized-Storage-Solution.pdf
