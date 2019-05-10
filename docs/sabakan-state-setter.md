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
  - serf tags `systemd-units-failed` has no errors or not set.
  - All of later mentioned machine peripherals are healthy.
- Set `Unreachable`
  - serf status is `failed`, `left` or machine is not yet as a serf member. It is the same as that `sabakan-state-setter` can not access monitor-hw metrics.
- Set `Unhealthy`
  - serf status is `alive`.
  - At least one of them matches:
    - serf tags `systemd-units-failed` has errors.
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

### `MachineType`

| Field                             | Default value            | Description                                                                                                               |
| -------------------               | ------------------------ | --------------                                                                                                            |
| `name` string                     | `''`                     | Name of this machine type. It is expected that this field is unique in setting file.                                      |
| `metrics` [Metric](#Metric) array | `nil`                    | Metrics is a array of `Metric`. If any of the metrics declared in this list are not healthy, then the machine is unhealthy. |

### `Metric`

| Field                        | Default value            | Description                                                                                                    |
| -------------------          | ------------------------ | --------------                                                                                                 |
| `name` string                | `''`                     | Name of this metric.                                                                                           |
| `labels` `map[string]string` | `nil`                    | Label map. The machine will be Unhealthy unless all labels defined in labels are present and not healthy. |

[serf]: https://www.serf.io/
[monitor-hw]: https://github.com/cybozu-go/setup-hw/blob/master/docs/monitor-hw.md
[Dell BOSS]: https://i.dell.com/sites/doccontent/shared-content/data-sheets/en/Documents/Dell-PowerEdge-Boot-Optimized-Storage-Solution.pdf
