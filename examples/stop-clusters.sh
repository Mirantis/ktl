#!/bin/sh
EXAMPLES_DIR=`dirname "$0"`
DOCKER=`command -v docker || command -v podman`

if [ -z "$DOCKER" ]; then
    echo "Can't find docker or podman"
    exit 1
fi

for CLUSTER_DIR in $EXAMPLES_DIR/import/*-cluster-*; do
    CLUSTER=`basename "$CLUSTER_DIR"`
    $DOCKER stop -t0 "$CLUSTER"
done
