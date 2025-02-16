#!/bin/sh

cd "`dirname "$0"`"

find ../../examples -type f -name '*.yaml' | grep -v rekustomization | xargs rm

./simple.sh
./clear.sh
./simple-filtered.sh
./clear.sh
./kubediff.sh
./clear.sh
./simple-helm.sh
./clear.sh
./simple-components.sh
