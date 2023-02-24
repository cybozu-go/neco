#!/bin/sh -ex

. $(dirname $0)/env
SUITE_NAME=$1
TAG_NAME=$2

# Run data center test
cat >run.sh <<EOF
#!/bin/sh -ex

# Set environment variables
GO111MODULE=on
export GO111MODULE
GOPATH=\${HOME}/go
export GOPATH
PATH=/usr/local/go/bin:\${GOPATH}/bin:\${PATH}
export PATH
if [ "${SUITE_NAME}" = "upgrade" ]; then
  MACHINES_FILE=/tmp/release/dctest/output/machines.yml
else
  MACHINES_FILE=\${GOPATH}/src/github.com/cybozu-go/neco/dctest/output/machines.yml
fi

# Run dctest
cd \${GOPATH}/src/github.com/cybozu-go/neco/dctest
exec make test TAGS=${TAG_NAME} SUITE=${SUITE_NAME} MACHINES_FILE=\${MACHINES_FILE}
EOF
chmod +x run.sh

$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo -H /home/cybozu/run.sh'
exit $?
