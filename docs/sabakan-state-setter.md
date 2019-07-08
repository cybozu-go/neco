sabakan-state-setter
====================

This command set sabakan machine states according to [serf][] status and [monitor-hw][] metrics.
It runs by systemd oneshot service which is executed periodically by `systemd.timer`.

Machine state selection
-----------------------

See machine state types at [Sabakan lifecycle management](https://github.com/cybozu-go/sabakan/blob/master/docs/lifecycle.md).

`sabakan-state-setter` decides machine state by the strategy as follows:

- Set `Healthy`
  - serf status is `alive`.
  - serf tags `systemd-units-failed` is set and has no errors.
  - All of later mentioned machine peripherals are healthy.
- Set `Unreachable`
  - serf status is `failed`, `left` or machine is not yet as a serf member. It is the same as that `sabakan-state-setter` can not access monitor-hw metrics.
- Set `Unhealthy`
  - serf status is `alive`.
  - At least one of them matches:
    - serf tags `systemd-units-failed` is not set or has errors.
    - `sabakan-state-setter` can not retrieve monitor-hw metrics.
    - At least one of later mentioned machine peripherals is unhealthy.
- Nothing to set machine state
  - `sabakan-state-setter` can not access to `serf.service` of the same boot server.

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

| Option              | Default value            | Description         |
| ------------------- | ------------------------ | --------------      |
| `--sabakan-address` | `http://localhost:10080` | URL of sabakan      |
| `--config-file`     | `''`                     | Path of config file |

Settings of target machine peripherals
--------------------------------------

The set of metrics used for health checking depends on its machine type.
You can configure a set of metrics to scrape for each machine type.

| Field                                             | Default value            | Description                                                                                         |
| -------------------                               | ------------------------ | --------------                                                                                      |
| `machine-types` [MachineType](#MachineType) array | `nil`                    | Machine types is a list of `MachineType`. You should list all machine types used in your data center. |

If all metrics defined in the config file are healthy, the machine is healthy. Otherwise, it's unhealthy.

### `MachineType`

| Field                             | Default value            | Description                                                                                                               |
| -------------------               | ------------------------ | --------------                                                                                                            |
| `name` string                     | `''`                     | Name of this machine type. It is expected that this field is unique in setting file.                                      |
| `metrics` [Metric](#Metric) array | `nil`                    | Metrics is an array of `Metric` to be checked. |

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
