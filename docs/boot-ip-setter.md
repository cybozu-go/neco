boot-ip-setter
==============

`boot-ip-setter` is a daemon program for handling virtual IP addresses on the boot servers.
It runs on active boot servers, where the neco package has been installed, and components such as etcd and sabakan, etc., are running.

This program handles the following two types of virtual IP addresses for different uses:

1. DHCP Server Address

    The IP address for Sabakan DHCP server. This address is used as the DHCP relay destination in the network switches.

    This program selects one of the five addresses from `10.71.255.1` to `10.72.255.5` and sets the address to its running server.
    If multiple active boot servers exist, these addresses will be set without bias to each server.

2. Active Boot Server Address

    The IP address for accessing one of the boot servers from inside the Kubernetes cluster.
    The value is fixed at `10.71.255.6`. The same value is set for all active boot servers.

This program decides whether or not to set these IPs based on the member list of the etcd cluster on boot servers and sets the IPs to the network interface.


## Usage (Options)

```console
$ boot-ip-setter [OPTIONS]
```

| Option          | Default value  | Description                                              |
| --------------- | -------------- | -------------------------------------------------------- |
| `-debug`        | `false`        | Show debug log or not.                                   |
| `-interface`    | `boot`         | The target network interface that this program operates. |
| `-interval`     | `1m`           | The interval for periodic operation.                     |
| `-listen-addr ` | `0.0.0.0:4192` | The listen address.                                      |


## HTTP endpoint

This program provides the following HTTP endpoints.

- `/hostname`

    This endpoint returns the hostname of the server that this program runs on.
    This is mainly intended for use in testing or operational checks.

- `/metrics`

    This endpoint returns the metrics. For details on metrics, please refer to the next section.


## Metrics

This program provides the following metrics in the Prometheus format.
Besides this, it also outputs the metrics collected in the `GoCollector` and the `ProcessCollector` of the [Prometheus Go client library](https://github.com/prometheus/client_golang).

| Name                                              | Description                                         | Type    | Labels              |
| ------------------------------------------------- | --------------------------------------------------- | ------- | ------------------- |
| `boot_ip_setter_hostname`                         | The hostname this program runs on.                  | Gauge   | `hostname`          |
| `boot_ip_setter_interface_address`                | The IP address set to the target interface.         | Gauge   | `interface`, `ipv4` |
| `boot_ip_setter_interface_operation_errors_total` | The number of times the interface operation failed. | Counter |                     |


## Internals

### Main process

This program repeats the following actions in one-minute cycles.

- Gets member list of the etcd cluster on boot servers.
- Calculates the virtual IPs should be set from the member list.
- Sets the IP address to the target network interface. If there are any unnecessary IPs on the interface, this program deletes them.

This program doesn't advertise the IPs, it just sets IPs to the network interfaces.

### Signal Handling

This program terminates normally when receiving `SIGTERM` or `SIGINT`.

### Error Handling

This program handles errors as follows.

- Connection failure to the etcd

    This program will terminate abnormally and delete the IPs on the target interface on exit.
    These errors may be resolved by retrying. So terminates early and retries from the beginning.

- Operation failure of the network interface

    This program will count up the `boot_ip_setter_interface_operation_errors_total` metric.
    These errors may not be recovered by restarting. So this program continues running and notifies errors by using metrics.

- Other failure

    If an error other than the above occurs, this program will terminate abnormally and delete the IPs on the target interface on exit.
