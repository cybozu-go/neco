#!/bin/bash

# ./neco-download-dir-cache.sh <operation>
# <operation> is a cache operation type.
#   fetch: fetch an archive from the cache and expand it to ${NECO_DIR}/download directory.
#   write: archive ${NECO_DIR}/download directory and write it to the cache if fetch was not succeeded.

# If NECO_CI_BLOB_CACHE_URL environment variable is set, do a cache operation for ${NECO_DIR}/download directory.
# Otherwise, this script do nothing.

if [ -z "${NECO_CI_BLOB_CACHE_URL}" ]; then
    echo not using cache
    exit 0
fi

md5str() {
    md5sum ${NECO_DIR}/$1 | cut -d' ' -f1
}

ROOTDIR=$(dirname $0)/..
CACHEURL=${NECO_CI_BLOB_CACHE_URL%/}/archive/neco-download/0-$(md5str Makefile)-$(md5str Makefile.tools)-$(md5str Makefile.common).tar.gz # The first section of the filename is cache format
LOCALNAME=/tmp/neco-download.tar.gz
COMPRESSOPT="-z"

# If the previously fetched (or pushed) archive exists, no need to access cache again.
if [ -e ${LOCALNAME} ]; then
    echo cache access skipped
    exit 0
fi

case "$1" in
    fetch)
        if curl -sSfL -o ${LOCALNAME} ${CACHEURL}; then
            echo cache hit
            tar -x -f ${LOCALNAME} -C ${ROOTDIR}
        else
            echo cache miss
        fi
        ;;
    write)
        tar -c -f ${LOCALNAME} -C ${ROOTDIR} ${COMPRESSOPT} download
        if curl -sSfL -T ${LOCALNAME} ${CACHEURL}; then
            echo cache write succeeded
        else
            echo cache write failed
        fi
        ;;
    *)
        exit 1
        ;;
esac
