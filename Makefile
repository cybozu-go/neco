# Makefile for neco

SUDO = sudo
FAKEROOT = fakeroot
ETCD_DIR = /tmp/neco-etcd
TAGS =

### for Go
GOFLAGS = -mod=vendor
GOTAGS = $(TAGS) containers_image_openpgp containers_image_ostree_stub
export GOFLAGS

BIN_PKGS = ./pkg/necogcp
STATIK = gcp/statik/statik.go

all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    start-etcd  - run etcd on localhost."
	@echo "    stop-etcd   - stop etcd."
	@echo "    test        - run single host tests."
	@echo "    mod         - update and vendor Go modules."
	@echo "    setup       - install dependencies."

start-etcd:
	systemd-run --user --unit neco-etcd.service etcd --data-dir $(ETCD_DIR)

stop-etcd:
	systemctl --user stop neco-etcd.service

$(STATIK):
	mkdir -p $(dir $(STATIK))
	go generate ./pkg/necogcp/...

test: $(STATIK)
	test -z "$$(gofmt -s -l . | grep -v '^vendor\|^menu/assets.go' | tee /dev/stderr)"
	test -z "$$(golint $$(go list -tags='$(GOTAGS)' ./... | grep -v /vendor/) | grep -v '/dctest/.*: should not use dot imports' | tee /dev/stderr)"
	go build -tags='$(GOTAGS)' ./...
	go test -tags='$(GOTAGS)' -race -v ./...
	RUN_COMPACTION_TEST=yes go test -tags='$(GOTAGS)' -race -v -run=TestEtcdCompaction ./worker/
	go vet -tags='$(GOTAGS)' ./...

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod

necogcp: $(STATIK)
	go install ./pkg/necogcp

clean:
	rm -rf $(ETCD_DIR) $(WORKDIR)

setup:
	GO111MODULE=off go get -u golang.org/x/lint/golint github.com/rakyll/statik

.PHONY:	all start-etcd stop-etcd test mod deb necogcp setup clean
