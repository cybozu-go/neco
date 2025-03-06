# Makefile for custom Ubuntu installer

SUDO = sudo

UBUNTU_VERSION = 22.04.5
UBUNTU_RELEASE = 22.04
UBUNTU_RELEASE_DATE = 20250228
BUILD_DEPS := xorriso qemu-utils qemu-kvm ovmf curl ca-certificates cloud-image-utils gdisk kpartx

ORIGINAL_ISO_NAME = ubuntu-$(UBUNTU_VERSION)-live-server-amd64.iso
ORIGINAL_ISO_PATH = build/$(ORIGINAL_ISO_NAME)
CUSTOM_ISO_PATH = build/cybozu-$(ORIGINAL_ISO_NAME)
SRC_DIR_PATH = build/src

ORIGINAL_CLOUD_IMAGE = ubuntu-$(UBUNTU_RELEASE)-server-cloudimg-amd64.img
ORIGINAL_CLOUD_PATH = build/$(ORIGINAL_CLOUD_IMAGE)
CUSTOM_CLOUD_PATH = build/cybozu-$(ORIGINAL_CLOUD_IMAGE)

PREVIEW_IMG = build/ubuntu.img
LOCALDS_IMG = build/seed.img

ifeq ($(shell lsb_release --release --short),$(UBUNTU_RELEASE))
# usage: get-deb-filename PACKAGE
define get-deb-filename
$(shell apt-get download --print-uris $1 | awk '{print $$2}')
endef
CHRONY_DEB = build/$(call get-deb-filename,chrony)
BIRD2_DEB = build/$(call get-deb-filename,bird2)
LIBSSH_DEB = build/$(call get-deb-filename,libssh-gcrypt-4)
# Though the "libssl3" package can be installed from the original ISO image, it could be old in terms of security.
# We download and burn the version with the security fixes applied.
LIBSSL3_DEB = build/$(call get-deb-filename,libssl3)
else
CHRONY_URL = http://archive.ubuntu.com/ubuntu/pool/main/c/chrony/chrony_4.2-2ubuntu2_amd64.deb
CHRONY_DEB = build/$(notdir $(CHRONY_URL))
CHRONY_SHA512 = 624f075f6415c71584404418d6cee32cb8498dcd809fe0d441497ce95f23b76c6ecb74e58afc8ccbeaa8c0af20740e103f7398d443e696971f12a4955a05abaf
BIRD2_URL = http://archive.ubuntu.com/ubuntu/pool/universe/b/bird2/bird2_2.0.8-2_amd64.deb
BIRD2_DEB = build/$(notdir $(BIRD2_URL))
BIRD2_SHA512 = 21b17a1e52dba90e0f35bb6b0cd8048c355de4c8364951d495e50d6e387ca807c4294e44444cd721b2cbf2d93e73f6824e748cf670b9cd0c99e2cc2f313f6f7e
LIBSSH_URL = http://archive.ubuntu.com/ubuntu/pool/main/libs/libssh/libssh-gcrypt-4_0.9.6-2build1_amd64.deb
LIBSSH_DEB = build/$(notdir $(LIBSSH_URL))
LIBSSH_SHA512 = c0a52a502da59cc644e178b351dc1edfdc44e04ea61184fc858e6d9dbdf161b27f2a30371e721a3ffb9c513ac23706fecc0df1b21259ec1183788049bf64547d
LIBSSL3_URL = http://security.ubuntu.com/ubuntu/pool/main/o/openssl/libssl3_3.0.2-0ubuntu1.19_amd64.deb
LIBSSL3_DEB = build/$(notdir $(LIBSSL3_URL))
LIBSSL3_SHA512 = 970230d32d610711466f7590245905ee9e7450da80c957d613266392ee925407c3d7c5018f11bf3f14676d9d075a67b406bb6da7f3f86828cad1a9d16bbabc5d
endif
DEBS = $(CHRONY_DEB) $(BIRD2_DEB) $(LIBSSH_DEB) $(LIBSSL3_DEB)

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
	../bin/download-with-blob-cache.sh "curl -fsSL" $@ https://releases.ubuntu.com/$(UBUNTU_VERSION)/$(ORIGINAL_ISO_NAME)

$(CHRONY_DEB):
	mkdir -p build
ifndef CHRONY_URL
	cd build; apt-get download chrony
else
	curl -o $@.tmp -fsSL $(CHRONY_URL)
	echo "$(CHRONY_SHA512) $@.tmp" | sha512sum -c -
	mv $@.tmp $@
endif

$(BIRD2_DEB):
	mkdir -p build
ifndef BIRD2_URL
	cd build; apt-get download bird2
else
	curl -o $@.tmp -fsSL $(BIRD2_URL)
	echo "$(BIRD2_SHA512) $@.tmp" | sha512sum -c -
	mv $@.tmp $@
endif

$(LIBSSH_DEB):
	mkdir -p build
ifndef LIBSSH_URL
	cd build; apt-get download libssh-gcrypt-4
else
	curl -o $@.tmp -fsSL $(LIBSSH_URL)
	echo "$(LIBSSH_SHA512) $@.tmp" | sha512sum -c -
	mv $@.tmp $@
endif

$(LIBSSL3_DEB):
	mkdir -p build
ifndef LIBSSL3_URL
	cd build; apt-get download libssl3
else
	curl -o $@.tmp -fsSL $(LIBSSL3_URL)
	echo "$(LIBSSL3_SHA512) $@.tmp" | sha512sum -c -
	mv $@.tmp $@
endif

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
	GOBIN=$(abspath $(SRC_DIR_PATH)/extras) CGO_ENABLED=0 go install -ldflags="-s -w" ./setup
	cp cluster.json $(SRC_DIR_PATH)/extras/

	# Setup autoinstall script
	cp -a autoinstall $(SRC_DIR_PATH)

	# Build an ISO file
	# See below url for the meaning of the xorriso command option
	# https://askubuntu.com/questions/1403546/ubuntu-22-04-build-iso-both-mbr-and-efi
	xorriso -as mkisofs -r -V "neco-ubuntu-$(UBUNTU_VERSION)" \
		-R -l -b '/boot/grub/i386-pc/eltorito.img' \
		-c /boot.catalog -no-emul-boot \
		-append_partition 2 28732ac11ff8d211ba4b00a0c93ec93b --interval:local_fs:2871452d-2879947d::'$(ORIGINAL_ISO_PATH)' \
		-e '--interval:appended_partition_2_start_1782357s_size_8496d:all::'  \
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
	../bin/download-with-blob-cache.sh "curl -fsSL" $@ https://cloud-images.ubuntu.com/releases/$(UBUNTU_RELEASE)/release-$(UBUNTU_RELEASE_DATE)/$(ORIGINAL_CLOUD_IMAGE)

$(CUSTOM_CLOUD_PATH): $(ORIGINAL_CLOUD_PATH) $(DEBS) cluster.json
	cp $< $@
	qemu-img resize $@ 30G
	GOBIN=$(abspath build) CGO_ENABLED=0 go install -ldflags="-s -w" ./setup
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
