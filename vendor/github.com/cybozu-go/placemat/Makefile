# Makefile for placemat

SUDO = sudo
FAKEROOT = fakeroot

### for Go
GOFLAGS = -mod=vendor
export GOFLAGS

### for debian package
PACKAGES := fakeroot
WORKDIR := $(CURDIR)/work
CONTROL := $(WORKDIR)/DEBIAN/control
DOCDIR := $(WORKDIR)/usr/share/doc/placemat
EXAMPLEDIR := $(WORKDIR)/usr/share/doc/placemat/examples
BASH_COMPLETION_DIR := $(WORKDIR)/etc/bash_completion.d
BINDIR := $(WORKDIR)/usr/bin
VERSION = 1.1.0-master
DEB = placemat_$(VERSION)_amd64.deb
DEST = .
SBIN_PKGS = ./pkg/placemat ./pkg/pmctl

test:
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	golint -set_exit_status $$(go list ./... | grep -v /vendor/)
	go build ./...
	go test -race -v ./...
	go vet ./...

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
	GOBIN=$(BINDIR) go install $(SBIN_PKGS)
	mkdir -p $(DOCDIR)
	cp README.md LICENSE docs/pmctl.md $(DOCDIR)
	cp -r examples $(DOCDIR)
	mkdir -p $(BASH_COMPLETION_DIR)
	$(BINDIR)/pmctl completion > $(BASH_COMPLETION_DIR)/placemat
	chmod -R g-w $(WORKDIR)
	$(FAKEROOT) dpkg-deb --build $(WORKDIR) $(DEST)
	rm -rf $(WORKDIR)

setup:
	GO111MODULE=off go get -u golang.org/x/lint/golint
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)

clean:
	rm -rf $(WORKDIR) $(DEB)

.PHONY:	all test mod deb setup clean
