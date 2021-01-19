#!/bin/sh -xe

TARGET="$1"
TAGS="$2"

PLACEMAT_PID=$(echo $(pgrep placemat) | tr " " ",")

while true; do
    #if pmctl pod show operation >/dev/null 2>&1; then break; fi
    ### temporary fix; "pmctl pod show operation" succeeds with pid == 0 if placemat is not running
    if pmctl pod show operation >/dev/null 2>&1; then
        if [ $(pmctl pod show operation | jq .pid) -ne 0 ]; then
            break
        fi
    fi
    ### end of temporary fix
    if ! ps -p $PLACEMAT_PID > /dev/null; then
        echo "FAIL: placemat is no longer working."
        exit 1;
    fi
    echo "preparing placemat..."
    sleep 1
done

go mod download
sudo -E nsenter -t $(pmctl pod show operation | jq .pid) -n env PATH=$PATH SUITE=$SUITE $GINKGO -focus="${TARGET}" -tags="${TAGS}" .
