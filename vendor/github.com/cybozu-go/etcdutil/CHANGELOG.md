# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [1.3.4] - 2019-10-24
### Changed
- Update golang 1.13.3 (#22)

## [1.3.3] - 2019-08-20
### Changed
- Update etcd client library as of [etcd-3.3.15](https://github.com/etcd-io/etcd/releases/tag/v3.3.15).

## [1.3.2] - 2019-08-19
### Changed
- Update etcd client library as of [etcd-3.3.14](https://github.com/etcd-io/etcd/releases/tag/v3.3.14).
- Revert #11 "Workaround for [etcd bug #9949](https://github.com/etcd-io/etcd/issues/9949)".

## [1.3.1] - 2018-11-19
### Changed
- Workaround for [etcd bug #9949](https://github.com/etcd-io/etcd/issues/9949).

## [1.3.0] - 2018-10-15
### Added
- AddPFlags method for github.com/spf13/pflag package.

## [1.2.2] - 2018-10-10
### Changed
- Update Go module dependencies (#9).

## [1.2.1] - 2018-10-10
### Changed
- Remove http://127.0.0.1:4001 from the default endpoints (#8).

## [1.2.0] - 2018-10-09
### Added
- Common command-line flags (#7).

## [1.1.0] - 2018-09-14
### Added
- Opt in to [Go modules](https://github.com/golang/go/wiki/Modules).

## 1.0.0 - 2018-09-03

This is the first release.

[Unreleased]: https://github.com/cybozu-go/etcdutil/compare/v1.3.4...HEAD
[1.3.4]: https://github.com/cybozu-go/etcdutil/compare/v1.3.3...v1.3.4
[1.3.3]: https://github.com/cybozu-go/etcdutil/compare/v1.3.2...v1.3.3
[1.3.2]: https://github.com/cybozu-go/etcdutil/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/cybozu-go/etcdutil/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/cybozu-go/etcdutil/compare/v1.2.2...v1.3.0
[1.2.2]: https://github.com/cybozu-go/etcdutil/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/cybozu-go/etcdutil/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/cybozu-go/etcdutil/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/cybozu-go/etcdutil/compare/v1.0.0...v1.1.0
