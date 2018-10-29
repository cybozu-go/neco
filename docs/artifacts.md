Artifacts
=========

Artifacts `artifacts.go` is a collection of components which are tested by [dctest](../dctest/).
It is given as a go file with structs to include neco binaries see details in [types.go](../types.go).

There are two artifacts files.

- `artifacts.go`: For developers. They can update it anytime using `generate-artifacts`.
- `artifacts_release.go`: For CI. The job updates it using `generate-artifacts --release`. The developers are PROHIBITED to modify it to prevent merge conficts in CI flow.
