Data Center Test(dctest)
========================

[dctest](dctest/) directory contains test suites to run integration
tests in a virtual data center environment.

Type of Test Suites
-------------------

There are two types of test suites.

1. functions

    This suite tests full set of functions of Neco.  Especially this includes
    upgrading test and joining/removing node test.

    Upgrading test uses `artifacts.go` as an initially-installed package,
    and `artifacts_new.go` as an upgrade-object package.

2. bootstrap

    This suite tests initial setup of Neco.  This does not include
    upgrading test nor joining/removing node test.

Each test has an entry point of test as `<suite>/suite_test.go`.


Synopsis
--------

[`Makefile`](Makefile) setup virtual data center environment and runs dctest.

* `make setup`

    Install dctest required components.

* `make clean`

    Delete generated files in `output/` directory.

* `make placemat`

    Run `placemat` in background by systemd-run to start virtual machines.

* `make stop`

    Stop `placemat`.

* `make test-light`

    Run _functions_ dctest.

* `make test`

    Run `placemat` and do _functions_ dctest.  Stop `placemat` at the end.

* `make test-release`

    Run `placemat` and do _bootstrap_ dctest using `artifacts_release.go`
    in the release branch.  Stop `placemat` at the end.

Options
-------

## `secrets` file

`neco sabakan-upload` supports uploading private container images where are in quay.io.
dctest runs `neco config set quay-username` and `neco config set quay-password` automatically when `secrets` file exists.
To upload private container images for sabakan, put quay.io password in `dctest/secrets`.
