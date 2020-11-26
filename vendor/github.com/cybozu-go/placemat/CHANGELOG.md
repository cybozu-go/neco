# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [1.5.3] - 2020-09-29

### Fixed

- Activate vhost-net. (#119)

## [1.5.2] - 2020-09-29

### Fixed

- Randomize MAC address for KVM NICs (#117)

## [1.5.1] - 2020-07-21

### Fixed

- Fix aio parameter for node volume devices when cache is specified (#115)

## [1.5.0] - 2020-07-20

### Added

- Add cache mode parameter for node volume devices (#113).
- Support creating node volume devices using raw format files (#113).
- Support creating node volume devices using LVs on host machine (#113).

## [1.4.0] - 2019-12-09

### Added

- Add stub HTTPS server for virtual BMC (#101).

## [1.3.9] - 2019-10-11

### Changed

- Add `iptables` rules for internal networking (#98).

## [1.3.8] - 2019-10-01

### Changed

- Use host CPU flags with `qemu -cpu host` for stability (#96).
- Replace yaml library (#94).

## [1.3.7] - 2019-07-26

### Added

- Add qemu option to use para-virtualized RNG for fast boot (#92).

## [1.3.6] - 2019-07-22

### Added

- Software TPM support (#91).

## [1.3.5] - 2019-03-15

### Added

- [`pmctl`](docs/pmctl.md) Add forward subcommand (#85).

## [1.3.4] - 2019-03-11

### Changed

- Wait resuming VMs after saving/loading snapshots (#83).

## [1.3.3] - 2019-03-04

### Changed

- Use formal import path for k8s.io/apimachinery (#82).

## [1.3.2] - 2019-02-18

### Changed

- [`pmctl`](docs/pmctl.md) Exit abnormally when failed to connect to server (#81).

## [1.3.1] - 2019-01-22

### Added

- [`pmctl`](docs/pmctl.md) Add snapshot list command. (#80)

## [1.3.0] - 2019-01-18

### Added

- [`pmctl`](docs/pmctl.md) Add snapshot subcommand. (#79)

## [1.2.0] - 2018-12-07

### Added

- [`pmctl`](docs/pmctl.md) Add completion subcommand. (#73)
- Release Debian Package. (#74)

### Changed

- Use fixed Debian image. (#72)

## [1.1.0] - 2018-11-06

### Added

- [`pmctl`](docs/pmctl.md) is a command-line client to control placemat.

### Removed

- `placemat-connect` as it is replaced by `pmctl`.

## [1.0.1] - 2018-10-23

### Changed

- Use cybozu-go/well instead of cybozu-go/cmd

## [1.0.0] - 2018-10-21

### Added

- Many things.  See git log.

[Unreleased]: https://github.com/cybozu-go/placemat/compare/v1.5.3...HEAD
[1.5.3]: https://github.com/cybozu-go/placemat/compare/v1.5.2...v1.5.3
[1.5.2]: https://github.com/cybozu-go/placemat/compare/v1.5.1...v1.5.2
[1.5.1]: https://github.com/cybozu-go/placemat/compare/v1.5.0...v1.5.1
[1.5.0]: https://github.com/cybozu-go/placemat/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/cybozu-go/placemat/compare/v1.3.9...v1.4.0
[1.3.9]: https://github.com/cybozu-go/placemat/compare/v1.3.8...v1.3.9
[1.3.8]: https://github.com/cybozu-go/placemat/compare/v1.3.7...v1.3.8
[1.3.7]: https://github.com/cybozu-go/placemat/compare/v1.3.6...v1.3.7
[1.3.6]: https://github.com/cybozu-go/placemat/compare/v1.3.5...v1.3.6
[1.3.5]: https://github.com/cybozu-go/placemat/compare/v1.3.4...v1.3.5
[1.3.4]: https://github.com/cybozu-go/placemat/compare/v1.3.3...v1.3.4
[1.3.3]: https://github.com/cybozu-go/placemat/compare/v1.3.2...v1.3.3
[1.3.2]: https://github.com/cybozu-go/placemat/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/cybozu-go/placemat/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/cybozu-go/placemat/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/cybozu-go/placemat/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/cybozu-go/placemat/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/cybozu-go/placemat/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/cybozu-go/placemat/compare/v0.1...v1.0.0
