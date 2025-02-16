#!/bin/sh

source "`dirname "$0"`/utils.sh"

typetext - <<EOF
kubediff --cluster=@dev-cluster-a,prod-cluster-a@\
 deploy simple-app
EOF
sleep 2
sendkey "G"
sleep 1
typetext ":qa"
