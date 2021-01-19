# Makefile for neco-test

# configuration variables
OUTPUT = ./output
PLACEMAT = /usr/bin/placemat
GINKGO = env GO111MODULE=on $(GOPATH)/bin/ginkgo --failFast -v
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

export PLACEMAT GINKGO SUITE MACHINES_FILE ADDRESS_POOLS_FILE

CUSTOM_UBUNTU = cybozu-ubuntu-20.04-server-cloudimg-amd64.img

# non-configuration variables
SABAKAN_DIR = $(OUTPUT)/sabakan
DHCP_JSON = $(SABAKAN_DIR)/dhcp.json
BMC_USER = $(SABAKAN_DIR)/bmc-user.json
DEBVER = 9999.99.99
GENERATED_DEB = $(abspath $(OUTPUT)/neco_$(DEBVER)_amd64.deb)
DOWNLOADED_DEB = $(abspath $(OUTPUT)/neco_release_amd64.deb)
ifneq ($(SUITE),bootstrap)
  DEB = $(GENERATED_DEB)
else ifdef DATACENTER
  DEB = $(DOWNLOADED_DEB)
else
  DEB = $(GENERATED_DEB)
endif
PLACEMAT_DEPS = $(OUTPUT)/$(CUSTOM_UBUNTU) $(BMC_USER) $(OUTPUT)/bmc.cert $(OUTPUT)/bmc.key
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
	cd ../installer; $(MAKE) cloud
	mkdir -p $(OUTPUT)
	mv ../installer/build/$(CUSTOM_UBUNTU) $@

$(OUTPUT)/bmc.cert $(OUTPUT)/bmc.key:
	openssl req -new -newkey rsa:4096 -days 3650 -nodes -x509 \
	-subj "/C=US/ST=Denial/L=Springfield/O=Dis/CN=*.www.example.com" \
	-keyout $(OUTPUT)/bmc.key  -out $(OUTPUT)/bmc.cert

# SABAKAN_CONTENTS
$(DHCP_JSON):
	cd ..; go generate ./menu
	cd ..; go install ./pkg/placemat-menu
	$(GOPATH)/bin/placemat-menu -f $(MENU) -o $(OUTPUT)

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

.PHONY: placemat
placemat: $(PLACEMAT_DEPS)
	sudo rm -rf $(PLACEMAT_DATADIR)
	sudo rm -rf $(VM_SHARED_DIR)
	sudo systemd-run --unit=placemat.service $(PLACEMAT) -bmc-cert $(abspath output/bmc.cert) -bmc-key $(abspath output/bmc.key) -enable-virtfs $(abspath output/cluster.yml)
	@echo 'Placemat is running.  To kill it, do "make stop".'

.PHONY: stop
stop:
	sudo systemctl stop placemat.service
	sudo systemctl disable placemat.service
	sudo rm /run/systemd/transient/placemat.service
	sudo systemctl daemon-reload
	sudo systemctl reset-failed placemat.service

.PHONY: clean
clean:
	rm -rf $(OUTPUT)

.PHONY: setup
setup:
	cd ../ && time -p $(MAKE) setup
	cd ../installer && time -p $(MAKE) setup
	go install github.com/onsi/ginkgo/ginkgo
