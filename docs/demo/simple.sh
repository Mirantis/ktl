#!/bin/sh

source "`dirname "$0"`/utils.sh"

# Let's start with a simple example.
typetext "nvim export-simple/rekustomization.yaml"
# We'll generate manifests for the simple-app namespace
# for our development cluster.
sleep 1
sendkeys ' tf'
sleep 1
typetext "rekustomize export ."
# Let's take a look what we got.
sleep 3
sendkey 'C-d'
sleep 1
sendkeys ' o'
sleep 0.5
sendkeys 'Pk'
sleep 1
# Here's the kustomization.yaml with all the resources
# that rekustomize found.
typetext 'k'
# First let's take a look at the simple app.
sleep 1
typetext 'fapp'
sleep 1
sendkey 'j'
sleep 1
sendkey 'j'
sleep 1
sendkey ' e'
sleep 0.5
typetext ':Cmdiff @cat\ */*deploy*.yaml,kubectl\ get\ deploy\ simple-app\ -oyaml@'
sleep 1
typetext 'Hkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk'
sleep 1
sendkeys ' ef'
sleep 0.5
sendkey 'C-w'
sleep 0.1
sendkey 'ENTER'
sleep 0.25
sendkey k
sleep 0.25
sendkey k
sendkey ENTER
sendkey j
sendkey P
sleep 1
sendkey j
sleep 1
typetext ':qa'

