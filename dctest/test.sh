#!/bin/sh -xe

TARGET="$1"
TAGS="$2"

SUDO_OPTION=-E
if [ "$SUDO" = "" ]; then
    SUDO_OPTION=""
fi

PLACEMAT_PID="$(pgrep --exact placemat2 | tr " " ",")"

while true; do
    if [ "$(ip netns | grep operation)" != "" ]; then
        break
    fi

    if ! ps -p $PLACEMAT_PID > /dev/null; then
        echo "FAIL: placemat is no longer working."
        exit 1;
    fi
    echo "preparing placemat..."
    sleep 1
done

(cd ..; go build ./...)
go test -c .
rm dctest.test
$SUDO $SUDO_OPTION ip netns exec operation env PATH=$PATH SUITE=$SUITE $GINKGO -focus="${TARGET}" -tags="${TAGS}" .
