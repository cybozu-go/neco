# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [2.0.1] - 2019-02-19

### Added
- Ignition template can list remote files (#139).

### Changed
- Fix a critical degradation in ignition template introduced in 2.0.0 (#140).

## [2.0.0] - 2019-02-18

### Added
- Ignition templates have `version` to specify Ignition spec version for rendering (#138).
- Arithmetic functions are available in Ignition templates (#137).

### Changed
- [Semantic import versioning](https://github.com/golang/go/wiki/Modules#semantic-import-versioning) for v2 has been applied.
- REST API for Ignition templates has been revamped breaking backward-compatibility (#138).
- Go client library has been changed for new Ignition template API (#138).

## [1.2.0] - 2019-02-13

### Added
- `Machine.Info` brings NIC configuration information (#136).  
    This new information is also exposed in GraphQL and REST API.
- `ipam.json` adds new mandatory field `node-gateway-offset` (#136).  
    Existing installations continue to work thanks to automatic data conversion.

### Changed
- GraphQL data type `BMCInfoIPv4` is renamed to `NICConfig`.

### Removed
- `dhcp.json` obsoletes `gateway-offset` field (#136).  
    The field is moved to `ipam.json` as `node-gateway-offset`.

## [1.1.0] - 2019-01-29

### Added
- [ignition] `json` template function to render objects in JSON (#134).

## [1.0.1] - 2019-01-28

### Changed
- Fix a regression in ignition template introduced in #131 (#133).

## [1.0.0] - 2019-01-28

### Breaking changes
- `ipam.json` adds new mandatory field `bmc-ipv4-gateway-offset` (#132).
- Ignition template renderer sets `.` as `Machine` instead of `MachineSpec` (#132).

### Added
- `Machine` has additional information field for BMC NIC configuration (#132).

## Ancient changes

See [CHANGELOG-0](./CHANGELOG-0.md).

[Unreleased]: https://github.com/cybozu-go/sabakan/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/cybozu-go/sabakan/compare/v1.2.0...v2.0.0
[1.2.0]: https://github.com/cybozu-go/sabakan/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/cybozu-go/sabakan/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/cybozu-go/sabakan/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/cybozu-go/sabakan/compare/v0.31...v1.0.0
