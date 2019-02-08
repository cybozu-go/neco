# Makefile for neco-test

# configuration variables
BOOT0 = 10.72.48.0
BOOT1 = 10.72.48.1
BOOT2 = 10.72.48.2
BOOT3 = 10.72.48.3
K8S_CTRLPLANE0 = 10.69.0.4
K8S_CTRLPLANE1 = 10.69.0.196
K8S_CTRLPLANE2 = 10.69.1.132
PLACEMAT = /usr/bin/placemat
GINKGO = env GO111MODULE=on GOFLAGS=-mod=vendor $(GOPATH)/bin/ginkgo --failFast -v
SUITE = ./bootstrap
ifeq ($(findstring /,$(SUITE)),)
  SUITE_PACKAGE = ./$(SUITE)
else
  SUITE_PACKAGE = $(SUITE)
endif
DATACENTER = staging
PLACEMAT_DATADIR = /var/scratch/placemat
VM_SHARED_DIR = /mnt/placemat
MENU = menu.yml

export BOOT0 BOOT1 BOOT2 BOOT3 PLACEMAT GINKGO SUITE_PACKAGE

CUSTOM_UBUNTU = cybozu-ubuntu-18.04-server-cloudimg-amd64.img
CUSTOM_UBUNTU_API_URL = https://api.github.com/repos/cybozu/neco-ubuntu/releases/latest
CUSTOM_UBUNTU_DISK_SIZE = 30G

# non-configuration variables
OUTPUT = ./output
SABAKAN_DIR = $(OUTPUT)/sabakan
OPERATION_SSH_DIR = $(OUTPUT)/operation-ssh
DHCP_JSON = $(SABAKAN_DIR)/dhcp.json
AUTOCONFIG = $(SABAKAN_DIR)/auto-config
BMC_USER = $(SABAKAN_DIR)/bmc-user.json
DEBVER = 9999.99.99
GENERATED_DEB = $(abspath $(OUTPUT)/neco_$(DEBVER)_amd64.deb)
DOWNLOADED_DEB = $(abspath $(OUTPUT)/neco_release_amd64.deb)
ifeq ($(notdir $(SUITE_PACKAGE)),functions)
  BASE_DEB = $(DOWNLOADED_DEB)
else
  BASE_DEB = $(GENERATED_DEB)
endif
### transient package
OLD_NEW_DEB = $(SABAKAN_DIR)/neco_0.0.2_amd64.deb
PLACEMAT_DEPS = $(OUTPUT)/$(CUSTOM_UBUNTU) $(AUTOCONFIG) $(BMC_USER) $(OPERATION_SSH_DIR) $(OLD_NEW_DEB)
TEST_DEPS = $(GENERATED_DEB) $(BASE_DEB)
SSH_PRIVKEY = $(realpath ./dctest_key)
BOB_PRIVKEY = $(realpath ./bob_key)
BOB_PUBKEY = $(realpath ./bob_key.pub)

export DEBVER GENERATED_DEB BASE_DEB SSH_PRIVKEY BOB_PRIVKEY BOB_PUBKEY

all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    placemat  - run VMs in placemat."
	@echo "    stop      - stop placemat."
	@echo "    test      - run test in placemat env."
	@echo "    setup     - install dependencies."
	@echo
	@echo "'make test' accepts SUITE, DATACENTER, and TAGS."

# CUSTOM_UBUNTU
$(CUSTOM_UBUNTU):
	curl -fSL -o $@ $$(curl -fsSL $(CUSTOM_UBUNTU_API_URL) | jq -r '.assets[] | select (.name=="$(CUSTOM_UBUNTU)") | .browser_download_url')

$(OUTPUT)/$(CUSTOM_UBUNTU): $(CUSTOM_UBUNTU)
	qemu-img resize $< $(CUSTOM_UBUNTU_DISK_SIZE)  # Do this here because CUSTOM_IMAGE is included in vmx-enabled image.
	mkdir -p $(OUTPUT)
	rm -f $@
	ln $< $@
	touch $@

# SABAKAN_CONTENTS
$(DHCP_JSON):
	$(GOPATH)/bin/placemat-menu -f $(MENU) -o $(OUTPUT)

$(AUTOCONFIG): auto-config $(DHCP_JSON)
	cp $< $@

$(BMC_USER): bmc-user.json $(DHCP_JSON)
	cp $< $@

# Other contents
$(GENERATED_DEB):
	mkdir -p $(OUTPUT)
	cd ..; $(MAKE) deb VERSION=$(DEBVER) DEST=$(abspath $(OUTPUT)) TAGS=$(TAGS)

$(DOWNLOADED_DEB):
	mkdir -p $(OUTPUT)
	cd ..; go install -mod=vendor ./pkg/download-neco-deb
	download-neco-deb $(DATACENTER) > $@

### always use empty TAGS because this target is used only for functions test suite
$(OLD_NEW_DEB):
	mkdir -p $(SABAKAN_DIR)
	cd ..; $(MAKE) deb VERSION=0.0.2 DEST=$(abspath $(SABAKAN_DIR))

# workaround for placemat-menu
$(OPERATION_SSH_DIR):
	mkdir -p $(OPERATION_SSH_DIR)
	cp dctest_key $(OPERATION_SSH_DIR)/boot_key

# Targets
test: $(TEST_DEPS)
	./test.sh "$(TARGET)"

placemat: $(PLACEMAT_DEPS)
	sudo rm -rf $(PLACEMAT_DATADIR)
	sudo rm -rf $(VM_SHARED_DIR)
	sudo systemd-run --unit=placemat.service $(PLACEMAT) -enable-virtfs $(abspath output/cluster.yml)
	@echo 'Placemat is running.  To kill it, do "make stop".'

stop:
	sudo systemctl stop placemat.service
	sudo systemctl disable placemat.service
	sudo rm /run/systemd/transient/placemat.service
	sudo systemctl daemon-reload
	sudo systemctl reset-failed placemat.service

clean:
	rm -rf $(OUTPUT)

fullclean: clean
	rm -rf $(CUSTOM_UBUNTU)

setup:
	cd ../ && $(MAKE) setup
	GO111MODULE=off	go get -d -u github.com/cybozu-go/placemat-menu
	GO111MODULE=off go get github.com/onsi/ginkgo/ginkgo
	cd $(GOPATH)/src/github.com/cybozu-go/placemat-menu; GO111MODULE=on $(MAKE)

.PHONY:	all test placemat stop clean fullclean setup
