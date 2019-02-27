# Makefile for neco

include Makefile.common

FAKEROOT = fakeroot
ETCD_DIR = /tmp/neco-etcd
TAGS =

### for Go
GOFLAGS = -mod=vendor
GOTAGS = $(TAGS) containers_image_openpgp containers_image_ostree_stub

### for debian package
PACKAGES := fakeroot btrfs-tools pkg-config libdevmapper-dev
VERSION = 0.0.1-master
DEST = .
DEB = neco_$(VERSION)_amd64.deb
BIN_PKGS = ./pkg/neco
SBIN_PKGS = ./pkg/neco-updater ./pkg/neco-worker ./pkg/sabakan-serf-handler

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
	test -z "$$(golint $$(go list $(GOFLAGS) -tags='$(GOTAGS)' ./... | grep -v /vendor/) | grep -v '/dctest/.*: should not use dot imports' | tee /dev/stderr)"
	go build $(GOFLAGS) -tags='$(GOTAGS)' ./...
	go test $(GOFLAGS) -tags='$(GOTAGS)' -race -v ./...
	RUN_COMPACTION_TEST=yes go test $(GOFLAGS) -tags='$(GOTAGS)' -race -v -run=TestEtcdCompaction ./worker/
	go vet $(GOFLAGS) -tags='$(GOTAGS)' ./...

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod

deb: $(DEB)

$(DEB):
	make -f Makefile.tools SUDO=$(SUDO)
	cp -r debian $(WORKDIR)
	mkdir -p $(WORKDIR)/src $(BINDIR) $(SBINDIR) $(SHAREDIR) $(DOCDIR)/neco
	sed 's/@VERSION@/$(patsubst v%,%,$(VERSION))/' debian/DEBIAN/control > $(CONTROL)
	GOBIN=$(BINDIR) go install -tags='$(GOTAGS)' $(BIN_PKGS)
	GOBIN=$(SBINDIR) go install -tags='$(GOTAGS)' $(SBIN_PKGS)
	cp etc/* $(SHAREDIR)
	cp -a ignitions $(SHAREDIR)
	cp README.md LICENSE $(DOCDIR)/neco
	chmod -R g-w $(WORKDIR)
	$(FAKEROOT) dpkg-deb --build $(WORKDIR)/deb $(DEST)

necogcp: $(STATIK)
	go install ./pkg/necogcp

setup:
	GO111MODULE=off go get -u golang.org/x/lint/golint github.com/rakyll/statik
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)

clean:
	rm -rf $(ETCD_DIR) $(WORKDIR) $(DEB)

.PHONY:	all start-etcd stop-etcd test mod deb necogcp setup clean
