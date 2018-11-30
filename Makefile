# Makefile for neco

SUDO = sudo
FAKEROOT = fakeroot
ETCD_DIR = /tmp/neco-etcd
TAGS =

### for Go
GOFLAGS = -mod=vendor
export GOFLAGS

### for debian package
PACKAGES := fakeroot
WORKDIR := $(CURDIR)/work
CONTROL := $(WORKDIR)/DEBIAN/control
DOCDIR := $(WORKDIR)/usr/share/doc/neco
BINDIR := $(WORKDIR)/usr/bin
SBINDIR := $(WORKDIR)/usr/sbin
SHAREDIR := $(WORKDIR)/usr/share/neco
VERSION = 0.0.1-master
DEST = .
DEB = neco_$(VERSION)_amd64.deb
BIN_PKGS = ./pkg/neco
SBIN_PKGS = ./pkg/neco-updater ./pkg/neco-worker ./pkg/sabakan-serf-handler ./pkg/setup-hw

all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    start-etcd  - run etcd on localhost."
	@echo "    stop-etcd   - stop etcd."
	@echo "    test        - run single host tests."
	@echo "    mod         - update and vendor Go modules."
	@echo "    deb         - build Debian package."
	@echo "    setup       - install dependencies."

start-etcd:
	systemd-run --user --unit neco-etcd.service etcd --data-dir $(ETCD_DIR)

stop-etcd:
	systemctl --user stop neco-etcd.service

test:
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	golint -set_exit_status $$(go list -tags='$(TAGS)' ./... | grep -v /vendor/)
	go build -tags='$(TAGS)' ./...
	go test -tags='$(TAGS)' -race -v ./...
	go vet -tags='$(TAGS)' ./...

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod go.sum

deb: $(DEB)

$(DEB):
	rm -rf $(WORKDIR)
	cp -r debian $(WORKDIR)
	sed 's/@VERSION@/$(patsubst v%,%,$(VERSION))/' debian/DEBIAN/control > $(CONTROL)
	mkdir -p $(BINDIR)
	GOBIN=$(BINDIR) go install -tags='$(TAGS)' $(BIN_PKGS)
	mkdir -p $(SBINDIR)
	GOBIN=$(SBINDIR) go install -tags='$(TAGS)' $(SBIN_PKGS)
	mkdir -p $(SHAREDIR)
	go install -tags='$(TAGS)' ./pkg/fill-asset-name
	fill-asset-name ignitions $(SHAREDIR)/ignitions
	mkdir -p $(DOCDIR)
	cp README.md LICENSE $(DOCDIR)
	chmod -R g-w $(WORKDIR)
	$(FAKEROOT) dpkg-deb --build $(WORKDIR) $(DEST)
	rm -rf $(WORKDIR)

setup:
	GO111MODULE=off go get -u golang.org/x/lint/golint
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)

clean:
	rm -rf $(ETCD_DIR) $(WORKDIR) $(DEB)

.PHONY:	all start-etcd stop-etcd test mod deb setup clean
