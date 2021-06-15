# Makefile for custom Ubuntu installer

SUDO = sudo

UBUNTU_VERSION = 20.04.1
BUILD_DEPS := xorriso qemu-utils qemu-kvm ovmf curl ca-certificates cloud-image-utils gdisk kpartx

ORIGINAL_ISO_NAME = ubuntu-$(UBUNTU_VERSION)-live-server-amd64.iso
ORIGINAL_ISO_PATH = build/$(ORIGINAL_ISO_NAME)
CUSTOM_ISO_PATH = build/cybozu-$(ORIGINAL_ISO_NAME)
SRC_DIR_PATH = build/src

ORIGINAL_CLOUD_IMAGE = ubuntu-20.04-server-cloudimg-amd64.img
ORIGINAL_CLOUD_PATH = build/$(ORIGINAL_CLOUD_IMAGE)
CUSTOM_CLOUD_PATH = build/cybozu-$(ORIGINAL_CLOUD_IMAGE)

PREVIEW_IMG = build/ubuntu.img
LOCALDS_IMG = build/seed.img

CHRONY_URL = http://archive.ubuntu.com/ubuntu/pool/main/c/chrony/chrony_3.5-6ubuntu6.2_amd64.deb
CHRONY_DEB = build/$(notdir $(CHRONY_URL))
BIRD2_URL = http://archive.ubuntu.com/ubuntu/pool/universe/b/bird2/bird2_2.0.7-2_amd64.deb
BIRD2_DEB = build/$(notdir $(BIRD2_URL))
LIBSSH_URL = http://archive.ubuntu.com/ubuntu/pool/main/libs/libssh/libssh-gcrypt-4_0.9.3-2ubuntu2.1_amd64.deb
LIBSSH_DEB = build/$(notdir $(LIBSSH_URL))
DEBS = $(CHRONY_DEB) $(BIRD2_DEB) $(LIBSSH_DEB)

.PHONY: help
help:
	@echo "Targets:"
	@echo "    setup         - install packages to build custom images."
	@echo "    all           - build both custom ISO and cloud images."
	@echo "    iso           - build custom Ubuntu server ISO image."
	@echo "    preview-iso   - run QEMU/KVM to test custom ISO image."
	@echo "    cloud         - build custom Ubuntu cloud image."
	@echo "    preview-cloud - run QEMU/KVM to test custom cloud image."
	@echo "    clean         - remove built images."
	@echo "    fullclean     - do clean + remove downloaded artifacts."

.PHONY: all
all: iso cloud

.PHONY: iso
iso: $(CUSTOM_ISO_PATH)

$(ORIGINAL_ISO_PATH):
	mkdir -p build
	curl -o $@ -fsL https://releases.ubuntu.com/$(UBUNTU_VERSION)/$(ORIGINAL_ISO_NAME)

$(CHRONY_DEB):
	mkdir -p build
	cd build; curl -O -fsL $(CHRONY_URL)

$(BIRD2_DEB):
	mkdir -p build
	cd build; curl -O -fsL $(BIRD2_URL)

$(LIBSSH_DEB):
	mkdir -p build
	cd build; curl -O -fsL $(LIBSSH_URL)

$(CUSTOM_ISO_PATH): $(ORIGINAL_ISO_PATH) $(DEBS) cluster.json
	rm -rf $(SRC_DIR_PATH)
	mkdir -p $(SRC_DIR_PATH)
	xorriso -osirrox on -indev $< -extract / $(SRC_DIR_PATH)

	# Patch files
	find $(SRC_DIR_PATH) | xargs chmod u+w
	rm -f $(SRC_DIR_PATH)/isolinux/boot.cat
	cp -a patch/grub.cfg $(SRC_DIR_PATH)/boot/grub/grub.cfg
	rm $(SRC_DIR_PATH)/md5sum.txt
	cd $(SRC_DIR_PATH); find . -type f -print0 | xargs -0 md5sum > /tmp/md5sum.txt
	mv /tmp/md5sum.txt $(SRC_DIR_PATH)

	# Add extra files
	mkdir $(SRC_DIR_PATH)/extras
	cp $(DEBS) $(SRC_DIR_PATH)/extras/
	GOBIN=$(abspath $(SRC_DIR_PATH)/extras) go install ./setup
	cp cluster.json $(SRC_DIR_PATH)/extras/

	# Setup autoinstall script
	cp -a autoinstall $(SRC_DIR_PATH)

	# Build an ISO file
	xorriso -as mkisofs -r -V "neco-ubuntu-$(UBUNTU_VERSION)" \
		-R -l -b isolinux/isolinux.bin \
		-c isolinux/boot.cat -no-emul-boot \
		-e boot/grub/efi.img \
		-eltorito-alt-boot \
		-boot-load-size 4 -boot-info-table \
		-isohybrid-gpt-basdat \
		-o $(CUSTOM_ISO_PATH) $(SRC_DIR_PATH)

.PHONY: preview-iso
preview-iso: $(CUSTOM_ISO_PATH)
	rm -f $(PREVIEW_IMG)
	qemu-img create -f qcow2 $(PREVIEW_IMG) 20G
	kvm -m 2G \
		-bios /usr/share/ovmf/OVMF.fd \
		-net nic -net nic \
		-drive file=$(PREVIEW_IMG) \
		-drive file=$(CUSTOM_ISO_PATH),media=cdrom

.PHONY: cloud
cloud: $(CUSTOM_CLOUD_PATH)

$(ORIGINAL_CLOUD_PATH):
	mkdir -p build
	curl -o $@ -fsL https://cloud-images.ubuntu.com/releases/20.04/release/$(ORIGINAL_CLOUD_IMAGE)

$(CUSTOM_CLOUD_PATH): $(ORIGINAL_CLOUD_PATH) $(DEBS) cluster.json
	qemu-img convert -O raw $< $@
	qemu-img resize -f raw $@ 30G
	GOBIN=$(abspath build) go install ./setup
	$(SUDO) ./customize-cloud-image $@ $(DEBS) build/setup cluster.json

.PHONY: preview-cloud
preview-cloud: $(CUSTOM_CLOUD_PATH)
	rm -f $(PREVIEW_IMG) $(LOCALDS_IMG)
	cloud-localds $(LOCALDS_IMG) cybozu.seed
	cp $(CUSTOM_CLOUD_PATH) $(PREVIEW_IMG)
	kvm -m 2G -net nic -net nic \
		-drive file=$(PREVIEW_IMG) \
		-drive file=$(LOCALDS_IMG),format=raw

.PHONY: clean
clean:
	rm -f $(CUSTOM_ISO_PATH) $(CUSTOM_CLOUD_PATH)
	rm -rf $(SRC_DIR_PATH) $(PREVIEW_IMG) $(LOCALDS_IMG)

.PHONY: fullclean
fullclean:
	rm -rf build

.PHONY: setup
setup:
	$(SUDO) apt-get -y install --no-install-recommends $(BUILD_DEPS)
