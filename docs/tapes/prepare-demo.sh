#!/bin/sh
mkdir -p `dirname "$0"`/data/{dev,test,all,groups}
cd `dirname "$0"`/data

kubectl config use-context dev-cluster-a
rekustomize export ./dev \
    --namespaces 'my*'

kubectl config use-context test-cluster-a
rekustomize export ./test \
    --namespaces 'my*' \
    --resources '!namespaces'
rekustomize export ./all \
    --namespaces 'my*' \
    --clusters '*'
rekustomize export ./groups \
    --namespaces 'my*' \
    --clusters 'dev=dev-*' \
    --clusters 'test=test-cluster-[ab]' \
    --clusters 'prod=prod-cluster-a,prod-cluster-b'
