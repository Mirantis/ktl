#!/bin/sh
set -e -o pipefail

EXAMPLES_DIR=`dirname "$0"`
DOCKER=`command -v docker || command -v podman`
KWOK_IMAGE="registry.k8s.io/kwok/cluster:v0.6.1-k8s.v1.30.4"

if [ -z "$DOCKER" ]; then
    echo "Can't find docker or podman"
    exit 1
fi

if [ -z "`command -v jq`" ]; then
    echo "Can't find jq"
    exit 1
fi

for CLUSTER_DIR in $EXAMPLES_DIR/import/*-cluster-*; do
    CLUSTER=`basename "$CLUSTER_DIR"`
    echo Starting cluster $CLUSTER

    $DOCKER run --rm -d -p 8080 --name "$CLUSTER" "$KWOK_IMAGE"
    CLUSTER_PORT=`$DOCKER inspect "$CLUSTER" | jq -r '.[0].NetworkSettings.Ports["8080/tcp"][0].HostPort'`
    kubectl config set-cluster "$CLUSTER" --server="127.0.0.1:$CLUSTER_PORT"
    kubectl config set-context "$CLUSTER" --cluster="$CLUSTER" --namespace=default
done

echo Waiting for clusters to start
# TODO: smarter wait
sleep 15

for CLUSTER_DIR in $EXAMPLES_DIR/import/*-cluster-*; do
    CLUSTER=`basename "$CLUSTER_DIR"`
    echo Applying test manifests to cluster $CLUSTER
    kubectl apply --cluster "$CLUSTER" -k "$EXAMPLES_DIR/import/common"
    kubectl apply --cluster "$CLUSTER" -k "$EXAMPLES_DIR/import/$CLUSTER"
done

kubectl config use-context dev-cluster-a
