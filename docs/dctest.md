Data Center Test(dctest)
========================

[dctest](dctest/) directory contains test suites to run integration
tests in a virtual data center environment.

Synopsis
--------

[`Makefile`](Makefile) setup virtual data center environment and runs dctest.

* `make setup`

    Install dctest required components.

* `make clean`

    Delete generated files in `output/` directory.

* `make placemat`

    Run `placemat` to start virtual machines. To stop placemat, please run `sudo pkill placemat`.

* `make test-light`

    Run dctest.

* `make test`

    Run equivalent to `make placemat` and `make test-light`.

Options
-------

## `secrets` file

`neco sabakan-upload` supports uploading private container images where are in quay.io.
dctest runs `neco config set quay-username` and `neco config set quay-password` automatically when `secrets` file exists.
To upload private container images for sabakan, put quay.io password in `dctest/secrets`.

## `neco-updater-token` file

`neco-updater` watches GitHub release without authentication by default. It would receive rate limits of the GitHub API.
dctest runs `neco config set github-token` automatically when `neco-updater-token` file exists.
