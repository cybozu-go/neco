#!/bin/sh

sudo -b sh -c "echo \$\$ >/tmp/placemat_pid$$; exec $PLACEMAT -loglevel error -enable-virtfs output/cluster.yml"
sleep 1
PLACEMAT_PID=$(cat /tmp/placemat_pid$$)
echo "placemat PID: $PLACEMAT_PID"

fin() {
    echo "-------------- boot-0: cloud-init.log"
    cat /mnt/placemat/boot-0/cloud-init.log
    echo "-------------- boot-0: cloud-init-output.log"
    cat /mnt/placemat/boot-0/cloud-init-output.log
    sudo kill $PLACEMAT_PID
    echo "waiting for placemat to terminate..."
    while true; do
        if [ -d /proc/$PLACEMAT_PID ]; then
            sleep 1
            continue
        fi
        break
    done
}
trap fin INT TERM HUP 0

while true; do
    child_pid=$(pgrep -P $PLACEMAT_PID)
    operation_pid=$(pgrep -P ${child_pid} -f operation)
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
operation_pid=$(pgrep -P ${child_pid} -f operation)

sudo -E nsenter -t ${operation_pid} -n sh -c "export PATH=$PATH; $GINKGO"
RET=$?

exit $RET
