#!/bin/bash

# upload-initial-kernel-image.sh <kernel-url> <initrd-url> <version>
# <version> is flatcar release version. It is used as ID for "sabactl images upload"

# This script is intended to be executed from dctest golang program.
# i.e. run in "operation" network namespace and run by root user.

# This script should be used for neco-apps bootstrap only.
# This script SHOULD NOT be used for neco CI because it makes some part of test for `neco init-data` be skipped.

set -e

KERNELURL=$1
INITRDURL=$2
IMAGEVER=$3

DCTESTDIR=$(dirname $0)
BINDIR=${DCTESTDIR}/../bin
TMPDIR=$(mktemp -d "${DCTESTDIR}/osimage.XXXXXXXX")
SSH_CONFIG=${DCTESTDIR}/ssh_config
IDENTITY_FILE=${DCTESTDIR}/dctest_key

set -x

# Download and cache access without a proxy from "core" network namespace.
# Accessing non-FQDN hosts with a proxy from "operation" namespace does not work for some reason.
ip netns exec core ${BINDIR}/download-with-blob-cache.sh 'curl -sSfL' ${TMPDIR}/kernel ${KERNELURL}
ip netns exec core ${BINDIR}/download-with-blob-cache.sh 'curl -sSfL' ${TMPDIR}/initrd ${INITRDURL}

# dcssh/dcscp cannot be used here. They use "sudo" and "ip netns exec", which are not necessary.
scp -F ${SSH_CONFIG} -i ${IDENTITY_FILE} ${TMPDIR}/kernel ${TMPDIR}/initrd boot-0:
ssh -F ${SSH_CONFIG} -i ${IDENTITY_FILE} boot-0 sabactl images upload ${IMAGEVER} kernel initrd
ssh -F ${SSH_CONFIG} -i ${IDENTITY_FILE} boot-0 sabactl images index

set +x

rm -rf ${TMPDIR} || true
