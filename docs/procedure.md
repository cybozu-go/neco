Update artifacts
================

Please update [artifacts](../artifacts.go) when you want to update its components.

Update repository to the latest state.

```console
$ cd $GOPATH/src/github.com/cybozu-go/neco
$ git checkout main
$ git pull
```

Build `generate-artifacs` and update `artifacts.go`.

```
$ go install ./pkg/generate-artifacts/
$ generate-artifacts > artifacts.go
```

Create a PR.

```
$ git commit -s -m "Update artifacts"
$ git neco review
```

After CI is finished, merge it.
