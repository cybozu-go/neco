#!/bin/sh -ex

. $(dirname $0)/env
SUITE_NAME=$1
TAG_NAME=$2

# Run data center test
cat >run.sh <<EOF
#!/bin/sh -e

# Set environment variables
export GO111MODULE=on
export GOPATH=\$HOME/go
export PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH

# Run dctest
cd \$GOPATH/src/github.com/cybozu-go/neco/dctest
exec make test TAGS=${TAG_NAME} SUITE=${SUITE_NAME}
EOF
chmod +x run.sh

$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo -H /home/cybozu/run.sh'
exit $?
