#!/bin/sh

source "`dirname "$0"`/utils.sh"

typetext "nvim export-components/rekustomization.yaml"
sleep 1
sendkeys ' tf'
sleep 1
typetext "rekustomize export ."
sleep 2
sendkey 'C-d'
sleep 0.5
sendkeys ' e'
sleep 0.5
sendkeys 'jjjjjPZ'
sleep 1
sendkeys 'j'
sleep 1
sendkeys 'jjj'
sleep 1
sendkeys 'jjjj'
sleep 1
sendkeys 'j'
sleep 1
sendkeys 'j'
sleep 1
sendkeys 'jjj'
sleep 1
sendkeys 'j'
sleep 1
sendkeys 'jj'
sleep 1
sendkeys 'z'
sleep 1
sendkeys 'k'
sleep 0.5
sendkey ENTER
sleep 0.5
sendkey 'j'
sleep 0.5
sendkey ENTER
sleep 0.5
sendkey 'j'
sleep 1
sendkey 'j'
sleep 0.5
sendkey ENTER
sleep 0.5
sendkey 'j'
sleep 1
typetext ':qa'

