# Makefile for neco-test

# configuration variables
SUDO = sudo
OUTPUT = ./output
PLACEMAT = /usr/bin/placemat2
GINKGO = env GO111MODULE=on $(GOPATH)/bin/ginkgo --fail-fast -v  --timeout=90m
SUITE = bootstrap
PLACEMAT_DATADIR = /var/scratch/placemat
VM_SHARED_DIR = /mnt/placemat
ifeq ($(MENU_ARG),)
  MENU = menu.yml
else
  MENU = $(MENU_ARG)
endif
MACHINES_FILE ?= $(abspath $(OUTPUT)/machines.yml)
ADDRESS_POOLS_FILE = $(abspath ./address_pools.yaml)
CURL_OPTIONS = -fsSL
CURL = curl $(CURL_OPTIONS)
CURL_GITHUB = ../bin/curl-github $(CURL_OPTIONS)
CT_VERSION = 0.9.3
CT := $(shell pwd)/bin/ct

export PLACEMAT GINKGO SUITE MACHINES_FILE ADDRESS_POOLS_FILE SUDO

CUSTOM_UBUNTU = cybozu-ubuntu-22.04-server-cloudimg-amd64.img
FLATCAR_IMAGE := flatcar_production_qemu_image.img

GITHUB_TOKEN_FILE = $(CURDIR)/../github-token
ifneq ("$(wildcard $(GITHUB_TOKEN_FILE))","")
  GITHUB_TOKEN := $(shell cat $(GITHUB_TOKEN_FILE))
  export GITHUB_TOKEN
endif

# non-configuration variables
SABAKAN_DIR = $(OUTPUT)/sabakan
DHCP_JSON = $(SABAKAN_DIR)/dhcp.json
BMC_USER = $(SABAKAN_DIR)/bmc-user.json
DEBVER = 9999.12.31-99999
GENERATED_DEB = $(abspath $(OUTPUT)/neco_$(DEBVER)_amd64.deb)
DOWNLOADED_DEB = $(abspath $(OUTPUT)/neco_release_amd64.deb)
ifneq ($(SUITE),bootstrap)
  DEB = $(GENERATED_DEB)
else ifdef DATACENTER
  DEB = $(DOWNLOADED_DEB)
else
  DEB = $(GENERATED_DEB)
endif
PLACEMAT_DEPS = $(OUTPUT)/$(CUSTOM_UBUNTU) $(OUTPUT)/$(FLATCAR_IMAGE) $(BMC_USER) $(OUTPUT)/chrony.ign
TEST_DEPS = $(DEB)
SSH_PRIVKEY = $(realpath ./dctest_key)
BOB_PRIVKEY = $(realpath ./bob_key)
BOB_PUBKEY = $(realpath ./bob_key.pub)

export DEBVER DEB SSH_PRIVKEY BOB_PRIVKEY BOB_PUBKEY

.PHONY: all
all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    placemat  - run VMs in placemat."
	@echo "    stop      - stop placemat."
	@echo "    test      - run test in placemat env."
	@echo "    setup     - install dependencies."
	@echo
	@echo "'make test' accepts SUITE, DATACENTER, and TAGS."

$(OUTPUT)/$(CUSTOM_UBUNTU):
	cd ../installer; cp cluster.json.example cluster.json
	cd ../installer; $(MAKE) cloud SUDO="$(SUDO)"
	mkdir -p $(OUTPUT)
	mv ../installer/build/$(CUSTOM_UBUNTU) $@

$(OUTPUT)/$(FLATCAR_IMAGE):
	mkdir -p $(OUTPUT)
	../bin/download-with-blob-cache.sh "$(CURL)" $@.bz2 https://stable.release.flatcar-linux.net/amd64-usr/current/$(FLATCAR_IMAGE).bz2
	bunzip2 -q $@.bz2

# SABAKAN_CONTENTS
$(DHCP_JSON):
	cd ..; go install ./pkg/placemat-menu
	$(shell go env GOPATH)/bin/placemat-menu -f $(MENU) -o $(OUTPUT)

$(OUTPUT)/chrony.ign: $(DHCP_JSON) $(CT)
	mkdir -p $(OUTPUT)
	$(CT) -in-file=$(OUTPUT)/chrony-ign.yml -pretty -out-file=$@

$(BMC_USER): bmc-user.json $(DHCP_JSON)
	cp $< $@

# Other contents
$(GENERATED_DEB):
	mkdir -p $(OUTPUT)
	cd ..; time -p $(MAKE) gcp-deb VERSION=$(DEBVER) DEST=$(abspath $(OUTPUT)) TAGS=$(TAGS)

$(DOWNLOADED_DEB):
	mkdir -p $(OUTPUT)
	cd ..; go install ./pkg/download-neco-deb
	download-neco-deb $(DATACENTER) > $@

# Targets
.PHONY: test
test: $(TEST_DEPS)
	./test.sh "$(TARGET)" "$(TAGS)"

.PHONY: run-placemat-inside-container
run-placemat-inside-container: $(PLACEMAT_DEPS)
	$(SUDO) rm -rf $(PLACEMAT_DATADIR)
	$(SUDO) rm -rf $(VM_SHARED_DIR)
	$(SUDO) cp $(OUTPUT)/bird*.conf /etc/bird
	$(SUDO) cp $(OUTPUT)/squid.conf /etc/squid
	$(SUDO) sh -c "$(PLACEMAT) $(abspath $(OUTPUT)/cluster.yml) > /var/log/placemat.log  2>&1" &
	for i in $$(seq 60); do \
		if ! pgrep --exact placemat2 > /dev/null; then \
			break; \
		fi; \
		sleep 1; \
	done
	@echo 'Placemat is running.  To kill it, do "make stop-placemat-inside-container".'

.PHONY: stop-placemat-inside-container
stop-placemat-inside-container:
	$(SUDO) ./stop_placemat.sh

.PHONY: placemat
placemat: $(PLACEMAT_DEPS)
	$(SUDO) rm -rf $(PLACEMAT_DATADIR)
	$(SUDO) rm -rf $(VM_SHARED_DIR)
	$(SUDO) cp $(OUTPUT)/bird*.conf /etc/bird
	$(SUDO) cp $(OUTPUT)/squid.conf /etc/squid
	$(SUDO) systemd-run --unit=placemat.service $(PLACEMAT) $(abspath output/cluster.yml)
	@echo 'Placemat is running.  To kill it, do "make stop".'

.PHONY: stop
stop:
	$(SUDO) systemctl stop placemat.service
	$(SUDO) systemctl disable placemat.service
	$(SUDO) rm /run/systemd/transient/placemat.service
	$(SUDO) systemctl daemon-reload
	$(SUDO) systemctl reset-failed placemat.service

.PHONY: clean
clean:
	rm -rf $(OUTPUT)
	rm -rf ./bin
	cd ../ && $(MAKE) clean

.PHONY: setup
setup:
	cd ../ && time -p $(MAKE) setup
	cd ../installer && time -p $(MAKE) setup
	go install github.com/onsi/ginkgo/v2/ginkgo

$(CT):
	mkdir -p bin
	$(CURL_GITHUB) -o ct https://github.com/flatcar/container-linux-config-transpiler/releases/download/v$(CT_VERSION)/ct-v$(CT_VERSION)-x86_64-unknown-linux-gnu
	chmod +x ct
	$(SUDO) mv ct $@
