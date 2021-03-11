#!/bin/sh -x

PLACEMAT_PID="$(pgrep --exact placemat2)"

if [ "$PLACEMAT_PID" = "" ]; then
    echo "placemat is not running."
    exit 0;
fi

kill -TERM "$PLACEMAT_PID"

while true; do
    if ! ps -p "$PLACEMAT_PID" > /dev/null; then
        break
    fi
    echo "wating for placemat to stop..."
    sleep 10
done
