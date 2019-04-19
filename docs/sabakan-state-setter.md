sabakan-state-setter
====================

This command set sabakan machine states according to [serf][] status and [monitor-hw][] metrics.
It runs by systemd oneshot service which is executed periodically by `systemd.timer`.

This command runs as follows.

1. Check the status of the serf members and the metrics of monitor-hw of all available nodes according to sabakan machines list.
2. If the serf status is alive and serf tags `systemd-units-failed` is ok, the command changes sabakan machines state to `Healthy`.
3. If the serf status is failed, the command changes sabakan machines state to `Unreachable`.
4. Otherwise, it changes sabakan machines state to `Unhealthy`.
5. If a machine peripheral has some problems, the command changes sabakan machines state to `Unhealthy`.
6. If a machine peripheral has no problem(or is back to normal), the command changes sabakan machines state to `Healthy`.

If `serf.service` in the local boot server is stopped or failed, the command does nothing to change states.

Usage
-----

```console
$ sabakan-state-setter [OPTIONS]
```

| Option              | Default value            | Description    |
| ------------------- | ------------------------ | -------------- |
| `--sabakan-address` | `http://localhost:10080` | URL of sabakan |

[serf]: https://www.serf.io/
[monitor-hw]: https://github.com/cybozu-go/setup-hw/blob/master/docs/monitor-hw.md
