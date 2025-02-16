#!/bin/sh

source "`dirname "$0"`/utils.sh"

typetext "nvim export-simple-filtered/rekustomization.yaml"
sleep 1
sendkeys ' tf'
sleep 1
typetext "rekustomize export ."
sleep 2
sendkey 'C-d'
sleep 0.5
sendkeys ' e'
sleep 0.5
sendkeys 'jjjjjP'
sleep 1
sendkeys 'kkk'
sendkey ENTER
sleep 0.5
sendkey j
sleep 1
typetext ':qa'
