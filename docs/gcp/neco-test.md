Behavior of the `neco-test` GCP project
=======================================

Overview
--------

![workflow](http://www.plantuml.com/plantuml/svg/ZP7BJiCm44Nt_eghY4MxY2jO6e6A0bIL45JyWUjCKuDZHtwWvky9xaDKBQBRyPbxnZFJ4AMFgJLCgkWyYaVgZEinUtY2x3giXfebvSf88M9MBT1vzq4g5lk6zsIH7vSzN5oJHwFZEYttO2XO2gHa81IPqtPmMaN30nENwWJAihp7Q0UE8N25aQhHg6uo56xHoaz2zVRwF9_TSJuvfF2-DQYpPin4fRqoLCtXR1RjSx_QJKbMBWtLqAriA1ioCXYWFCb8-4KnyU_71JZ_A_gQuqKcgzROxMIx4dQEpYd7gniBLDkHZkj8GTi69o4NJLkMfnu8t70Scd_E6DZX2cTR11RajQkragP7r7MFr46lG1iTOk1iIhPEhVa6)

Test environment for [dctest][] and multi host test of other github Neco projects are provisioned by GCE.
All resources are in neco-test GCP project. Instance name is called by CircleCI which is based on `vmx-enabled` instance image.

GAE app
-------

`/shutdown` for `neco-test` project does:

- If instances have `shutdown-at` metadata and exceed that timestamp, delete them.
- If instances do not have `shutdown-at` metadata and run longer than configured lifetime (`app.shutdown.expiration`), delete them.

Cron is scheduled to run above actions every 5 minutes.

Usage
-----

### Edit Configuration file for neco-test

Edit [config.go](../../gcp/config.go)

### Update `vmx-enabled` installed packages for neco-test

Edit [artifacts.go](../../gcp/artifacts.go)

### Deploy GAE app

`google-cloud-sdk-app-engine-go` should be installed before the deployment.

```console
make -f Makefile.gcp PROJECT=neco-test create
```

### Create or Update `vmx-enabled` image for neco-test

```console
necogcp neco-test create-image
```

### Extend 2 hours from now to prevent auto deletion

```console
necogcp neco-test extend INSTANCE_NAME
```

[dctest]: https://github.com/cybozu-go/neco/tree/master/dctest
