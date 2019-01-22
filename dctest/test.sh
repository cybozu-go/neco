#!/bin/sh

sudo -b sh -c "echo \$\$ >/tmp/placemat_pid$$; exec $PLACEMAT -loglevel error -enable-virtfs output/cluster.yml"
sleep 1
PLACEMAT_PID=$(cat /tmp/placemat_pid$$)
echo "placemat PID: $PLACEMAT_PID"

fin() {
    if [ "$RET" -ne 0 ]; then
        # do not kill placemat upon test failure to help debugging.
        return
    fi
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

sudo -E nsenter -t ${operation_pid} -n sh -c "export PATH=$PATH; $GINKGO $SUITE"
RET=$?
if [ "$RET" -ne 0 ]; then
  exit $RET
fi

if [ "${SAVE_SNAPSHOT}" = "true" ]; then
    set -e

    VOLUMES=$(gsutil ls -d ${GCS_SNAPSHOT_BUCKET}/volumes_*)
    NOW=$(date "+%Y%m%d%H%M%S")
    NODES=$(${PMCTL} node list)

    ${PMCTL} snapshot save latest

    count=$(${PMCTL} snapshot list | jq '.[] | select(. | contains("There is no snapshot available."))' | wc -l)
    if [ $count -ne 0 ]; then
        echo "snapshots were not saved correctly"
        exit 1
    fi
    for node in ${NODES}; do
        ${PMCTL} node action stop ${node}
    done

    gsutil -q cp -r ${PLACEMAT_DATADIR}/volumes ${GCS_SNAPSHOT_BUCKET}/volumes_${NOW}_temp
    gsutil -q mv ${GCS_SNAPSHOT_BUCKET}/volumes_${NOW}_temp/* ${GCS_SNAPSHOT_BUCKET}/volumes_${NOW}
    gsutil rm -r ${GCS_SNAPSHOT_BUCKET}/volumes_${NOW}_temp
    for v in ${VOLUMES}; do
        gsutil rm -r $v
    done
fi

exit 0
