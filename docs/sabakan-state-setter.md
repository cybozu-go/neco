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
  - serf tags `systemd-units-failed` has no errors.
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

Usage
-----

```console
sabakan-state-setter [OPTIONS]
```

| Option              | Default value            | Description    |
| ------------------- | ------------------------ | -------------- |
| `--sabakan-address` | `http://localhost:10080` | URL of sabakan |

[serf]: https://www.serf.io/
[monitor-hw]: https://github.com/cybozu-go/setup-hw/blob/master/docs/monitor-hw.md
[Dell BOSS]: https://i.dell.com/sites/doccontent/shared-content/data-sheets/en/Documents/Dell-PowerEdge-Boot-Optimized-Storage-Solution.pdf
