#!/bin/sh

TARGET="$1"
TAGS="$2"

PLACEMAT_PID=$(echo $(pgrep placemat) | tr " " ",")

while true; do
    operation_pid=$(pgrep -P $PLACEMAT_PID -f operation)
    if sudo -E nsenter -t ${operation_pid} -n /bin/true 2>/dev/null; then break; fi
    if ! ps -p $PLACEMAT_PID > /dev/null; then
        echo "FAIL: placemat is no longer working."
        exit 1;
    fi
    echo "preparing placemat..."
    sleep 1
done

# obtain operation pod's pid again, because rkt's pid may change.
sleep 3
operation_pid=$(pgrep -P $PLACEMAT_PID -f operation)

sudo -E nsenter -t ${operation_pid} -n sh -c "export PATH=$PATH; $GINKGO -focus=\"${TARGET}\" -tags=\"${TAGS}\" $SUITE_PACKAGE"
