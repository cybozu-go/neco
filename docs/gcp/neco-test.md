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

- Delete instances which are created more than 1 hour ago.
- Set cron schedule to run above action every hour.

Usage
-----

### Configuration file for neco-test project

See [neco-test.yml](../../gcp/neco-test.yml)

### Deploy GAE app and `vmx-enabled` image

CircleCI updates them automatically.

[dctest]: https://github.com/cybozu-go/neco/tree/master/dctest
