# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [0.27] - 2018-11-27

### Changed
- Improve GraphQL implementation (#118).
- Improve client package (#119).

## [0.26] - 2018-11-27

### Changed
- Update version.go to release correctly.

## [0.25] - 2018-11-27

### Added
- GraphQL API and playground (#117)
- `spec.register-date` and `spec.retire-date` fields to `Machine` (#116).
- REST API to edit `retire-date` (#116).
- `status.duration` field to `Machine` (#116).

### Changed
- Update etcdutil to 1.3.1.

## [0.24] - 2018-10-25

### Changed
- Fix version.go

## [0.23] - 2018-10-25

### Added
- Add metadata field for ignitions and assets (#110, #111, #112, #113)

## [0.22] - 2018-10-11

### Changed
- Update machines lifecycle (#109).

## [0.21] - 2018-10-09

### Changed
- Update machines state (#106, #107).

### Added
- Add API `/api/v1/labels` (#108).

## [0.20] - 2018-09-03

### Changed
- Rebuild with latest etcdutil v1.0.0.

## [0.19] - 2018-08-24

### Added
- Support healthcheck and version endpoint (#97).

### Changed
- Support label on machines (#98).
- Modify BMC Type validation (#99).

## [0.18] - 2018-08-14

### Added
- Support configurable kernel parameters on booting CoreOS (#95).

## [0.17] - 2018-08-06

### Added
- Support for TLS client authentication for etcd (#93, #94).

## [0.16] - 2018-08-01

### Changed
- Fix infinite loop in asset updater after etcd compaction (#92).

## [0.15] - 2018-07-25

### Added
- Add integration tests using [placemat][] VMs.

## [0.14] - 2018-07-18

### Changed
- Fix a bug that leaves files of deleted OS images (#86).

[placemat]: https://github.com/cybozu-go/placemat
[Unreleased]: https://github.com/cybozu-go/sabakan/compare/v0.27...HEAD
[0.27]: https://github.com/cybozu-go/sabakan/compare/v0.26...v0.27
[0.26]: https://github.com/cybozu-go/sabakan/compare/v0.25...v0.26
[0.25]: https://github.com/cybozu-go/sabakan/compare/v0.24...v0.25
[0.24]: https://github.com/cybozu-go/sabakan/compare/v0.23...v0.24
[0.23]: https://github.com/cybozu-go/sabakan/compare/v0.22...v0.23
[0.22]: https://github.com/cybozu-go/sabakan/compare/v0.21...v0.22
[0.21]: https://github.com/cybozu-go/sabakan/compare/v0.20...v0.21
[0.20]: https://github.com/cybozu-go/sabakan/compare/v0.19...v0.20
[0.19]: https://github.com/cybozu-go/sabakan/compare/v0.18...v0.19
[0.18]: https://github.com/cybozu-go/sabakan/compare/v0.17...v0.18
[0.17]: https://github.com/cybozu-go/sabakan/compare/v0.16...v0.17
[0.16]: https://github.com/cybozu-go/sabakan/compare/v0.15...v0.16
[0.15]: https://github.com/cybozu-go/sabakan/compare/v0.14...v0.15
[0.14]: https://github.com/cybozu-go/sabakan/compare/v0.13...v0.14
