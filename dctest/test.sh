#!/bin/sh

TARGET="$1"
TAGS="$2"

PLACEMAT_PID=$(echo $(pgrep placemat) | tr " " ",")

while true; do
    if pmctl pod show operation >/dev/null 2>&1; then break; fi
    if ! ps -p $PLACEMAT_PID > /dev/null; then
        echo "FAIL: placemat is no longer working."
        exit 1;
    fi
    echo "preparing placemat..."
    sleep 1
done

sudo -E nsenter -t $(pmctl pod show operation | jq .pid) -n sh -c "export PATH=$PATH; $GINKGO -focus=\"${TARGET}\" -tags=\"${TAGS}\" $SUITE_PACKAGE"
