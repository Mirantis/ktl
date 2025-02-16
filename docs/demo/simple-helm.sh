#!/bin/sh

source "`dirname "$0"`/utils.sh"

typetext "nvim export-helm/rekustomization.yaml"
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
sendkeys 'jj'
sleep 1
sendkeys 'j'
sleep 1
sendkeys 'j'
sleep 1
sendkeys 'j'
sleep 1
sendkeys 'j'
sleep 1
sendkeys 'j'
sleep 1
sendkeys 'j'
sleep 1
sendkeys 'jjj'
sleep 1
sendkeys 'jj'
sleep 1
sendkeys 'jj'
sleep 1
sendkeys 'jj'
sleep 1
sendkeys 'jj'
sleep 1
typetext ':qa'

