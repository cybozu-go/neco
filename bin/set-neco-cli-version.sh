#!/bin/bash -e

# This is an auxiliary script called from Makefile.
# This does much the same as the following in Makefile.
#   sed -i -e "s/.../$(shell curl ...)/" FILE
# The status of the shell function is hard to check, so use this script.

set -o pipefail
NECO_CLI_VERSION=$($(dirname $0)/curl-github -sSfL "https://api.github.com/repos/cybozu-go/neco/releases/latest" | jq -r ".tag_name" | sed -e "s/release-//")
sed -i -e "s/@NECO_CLI_VERSION/${NECO_CLI_VERSION}/" $1
