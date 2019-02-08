Behavior of the `neco-test` GCP project
=======================================

Overview
--------

![workflow](http://www.plantuml.com/plantuml/svg/ZP3DJiCm48JlVefHn84U70cEFg2YW2ea3XLlu6wsgL5YM_OQDMyFIdz4r29wzNmpuzbb5fka3Bso926XUk7OXn6hvrVO6B4D2PufQE4iM3Lhn1G-cQGw6JwAnoHkHWJPSKBdP5Ss9p1NgcGccndLn3cVnNhY7q6PM-iCjDPFk3-22nZSJMH7SN9IOYkiJECIzToy8VX9Fnc_XhrcRpSzjt23xNWUGM68HVWOWr-qClykDAZhloeUQhpRucc7u_Z3TdMDdbBcDreOD8SlpTzHilCTBa9k-gtMbpqmUAnnDfFDdDNvt5Sj1cjEBhIER3z2N3kYHBWjUE-ov5ejsRTbwBy1)

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
