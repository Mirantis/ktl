#!/bin/sh

sendkey() {
  sleep 0.02
  echo -n "$1"
  if [ "$1" == $'\n' ]; then
    echo "<Enter>"
    sleep 1
  fi
  tmux send-keys -t demo "$1"
}

sendkeys() {
  echo -n "$1" | while IFS='' read -rN1; do
    sendkey "$REPLY"
  done
}

typetext() {
  if [ "$1" != "-" ]; then
    echo "$1"
  else
    cat
  fi | while IFS='' read -rN1; do
    sendkey "$REPLY"
  done
}

clearscreen() {
  typetext "clear"
}
