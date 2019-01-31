Data Center Test (dctest)
=========================

[dctest](dctest/) directory contains test suites to run integration
tests in a virtual data center environment.

Generation of deb Package
-------------------------

Dctest uses a generated deb package to install Neco.
Details of artifacts definition used in generation of the deb package are
described in [Artifacts](artifacts).

Type of Test Suites
-------------------

There are two types of test suites.

1. functions

    This suite tests full set of functions of Neco.  Especially this includes
    upgrading test and joining/removing node test.

    Upgrading test installs Neco with the deb package of the latest release
    in github.com at first, and then upgrades Neco to the generated deb package.

2. bootstrap

    This suite tests initial setup of Neco.  This does not include
    upgrading test nor joining/removing node test.

    This suite installs Neco with the generated deb package.

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

    Run dctest on a running `placemat`.  This does not contol `placemat` by itself.

* `make test`

    Run `placemat` and do dctest.  Stop `placemat` at the end.

Options
-------

## `SUITE`

You can choose the type of test suite by specifying `SUITE` make variable.
The default is to run `functions`.

Because this choice affects setup of placemat, specify this for `make placemat`
as well as for `make test`/`make test-light`.

## `TAGS`

You can choose the list of artifacts by specifying `TAGS` make variable.
The default is "" to use `artifacts.go`.
Specify "release" to use `artifacts_release.go`.

Because this choice affects setup of placemat, specify this for `make placemat`
as well as for `make test`/`make test-light`.

## `secrets` file

`neco sabakan-upload` supports uploading private container images where are in quay.io.
dctest runs `neco config set quay-username` and `neco config set quay-password` automatically when `secrets` file exists.
To upload private container images for sabakan, put quay.io password in `dctest/secrets`.
