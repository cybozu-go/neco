Update artifacts
================

Please update [artifacts](../artifacts.go) when you want to update its components.

Update repository to the latest state.

```console
$ cd $GOPATH/src/github.com/cybozu-go/neco
$ git checkout main
$ git pull
```
 
Or clone repository.

```console
$ git clone git@github.com:cybozu-go/neco.git
$ cd neco
```

Create branch.

```console
$ VERSION=x.y.z
$ echo $VERSION
$ git switch main
$ git pull origin main
$ git switch -c "bump-$VERSION"
```

Build `generate-artifacs` and update `artifacts.go`.

```console
$ go install ./pkg/generate-artifacts/
$ generate-artifacts > artifacts.go
$ git add artifacts.go
```

Create a PR.

```console
$ git commit -s -m "Update artifacts"
$ gh pr create --fill
```

After CI is finished, merge it.
