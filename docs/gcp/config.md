Configuration file `necogcp.yml`
=================================

Example is [necogcp-example.yml](necogcp-example.yml).

`necogcp.yml` is YAML formatted file for building GCE instances and a GAE application.

`common`
--------

Fields are used for all scripts in neco-gcp.

| Field            | Type   | Default | Description                                            |
| ---------------- | ------ | ------- | ------------------------------------------------------ |
| `project`        | string |         | GCP project                                            |
| `serviceaccount` | string |         | Account for `project` (can use your GCP login account) |
| `zone`           | string |         | GCP region where instances and images saved in         |

`app`
-----

Fields in `shutdown` are configuration for GAE endpoint [`/shutdown`](api.md#shutdown).

| Field              | Type          | Default      | Description                                                                                                 |
| ------------------ | ------------- | ------------ | ----------------------------------------------------------------------------------------------------------- |
| `stop`             | []string      |              | Target instances to be stopped by cron                                                                      |
| `exclude`          | []string      |              | Exclude instances to avoid shutdown/delete                                                                  |
| `expiration`       | time.Duration | `0s`         | Delete instances which are created `expiration` seconds ago. If `0s`, the instances are deleted immediately |
| `timezone`         | string        | `Asia/Tokyo` | Timezone of `shutdown-at`                                                                                   |
| `shutdown-at`      | string        | `20:00`      | Time of shutdown with hh:mm format                                                                          |
| `additional-zones` | []string      |              | Additional zones to watch instances                                                                         |

`compute`
---------

Fields are common configuration for GCE provisioning.

| Field               | Type     | Default | Description                   |
| ------------------- | -------- | ------- | ----------------------------- |
| `machine-type`      | string   |         | Instance machine type         |
| `boot-disk-sizeGB`  | int      | `20`    | Root filesystem size in GB    |
| `optional-packages` | []string | `[]`    | List of optional APT packages |

Fields in `auto-shutdown` are configuration for deleting `host-vm` instance.

| Field         | Type   | Default      | Description                        |
| ------------- | ------ | ------------ | ---------------------------------- |
| `timezone`    | string | `Asia/Tokyo` | Timezone of `shutdown-at`          |
| `shutdown-at` | string | `21:00`      | Time of shutdown with hh:mm format |

Fields in `host-vm` are configuration for `host-vm` instance.

| Field              | Type | Default | Description                                                                                                                                                              |
| ------------------ | ---- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `home-disk`        | bool | false   | Attach home disk to host-vm instance                                                                                                                                     |
| `home-disk-sizeGB` | int  | `20`    | Home disk size in GB. If you change bigger size than current size, the existing home disk is expanded. If it's expanded, please run `resize2fs` to expand the filesystem |
| `preemptible`      | bool | false   | Enable [`preemptible`](https://cloud.google.com/compute/docs/instances/preemptible)                                                                                      |
