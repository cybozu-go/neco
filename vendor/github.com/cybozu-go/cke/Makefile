# Makefile for cke

GOFLAGS = -mod=vendor
export GOFLAGS

all: test

setup:
	go install github.com/gostaticanalysis/nilerr/cmd/nilerr
	go install github.com/cybozu/neco-containers/golang/restrictpkg/cmd/restrictpkg

test:
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	test -z "$$(golint $$(go list ./... | grep -v /vendor/) | grep -v '/mtest/.*: should not use dot imports' | tee /dev/stderr)"
	test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	test -z "$$(restrictpkg -packages=html/template,log ./... 2>&1 | tee /dev/stderr)"
	ineffassign .
	go install ./pkg/...
	go test -race -v ./...
	go vet ./...

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod

.PHONY:	all setup test mod
