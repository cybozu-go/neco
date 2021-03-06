generate-artifacts
==================

`generate-artifacts` is a command-line tool to generate `artifacts.go`
which is a collection of latest components. This is used while
development or CI before build neco binaries.

See [artifacts.md](artifacts.md) about `artifacts.go`.

Usage
-----

```console
$ generate-artifacts [OPTIONS]
```

| Option      | Default value | Description                         |
| ----------- | ------------- | ----------------------------------- |
| `--release` | false         | Generate for `artifacts_release.go` |

Environment variables
---------------------

`generate-artifacts` uses GitHub personal token if `GITHUB_TOKEN` environment is set.
