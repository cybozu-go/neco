# Makefile for sabakan

GO_FILES=$(shell find -name '*.go' -not -name '*_test.go')
BUILT_TARGET=sabakan sabactl
ETCD_DIR = /tmp/neco-etcd

# for Go
GOFLAGS = -mod=vendor
export GOFLAGS

build: $(BUILT_TARGET)
$(BUILT_TARGET): $(GO_FILES)
	go build ./pkg/$@

start-etcd:
	systemd-run --user --unit neco-etcd.service etcd --data-dir $(ETCD_DIR)

stop-etcd:
	systemctl --user stop neco-etcd.service

test: build
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	golint -set_exit_status $$(go list ./... | grep -v /vendor/)
	go test -v -race -count=1 ./...
	go vet ./...

e2e: build
	RUN_E2E=1 go test -v -count=1 ./e2e

clean:
	rm -f $(BUILT_TARGET)

.PHONY: build start-etcd stop-etcd test e2e clean
