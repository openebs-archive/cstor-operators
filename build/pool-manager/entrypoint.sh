#!/bin/bash

set -ex

#sigterm caught SIGTERM signal and forward it to pool_manager process
_sigterm() {
  echo "[entrypoint.sh] caught SIGTERM signal forwarding to pid [$pool_manager_pid]."
  kill -TERM "$pool_manager_pid" 2> /dev/null
  waitForChildProcessToFinish
}

#sigint caught SIGINT signal and forward it to pool_manager process
_sigint() {
  echo "[entrypoint.sh] caught SIGINT signal forwarding to pid [$pool_manager_pid]."
  kill -INT "$pool_manager_pid" 2> /dev/null
  waitForChildProcessToFinish
}

#waitForChildProcessToFinish waits for pool_manager process to finish
waitForChildProcessToFinish(){
    while ps -p "$pool_manager_pid" > /dev/null; do sleep 1; done;
}

rm /usr/local/bin/zrepl
/usr/local/bin/pool-manager start &
pool_manager_pid=$!

#exec service ssh start
#exec service rsyslog start

trap '_sigint' INT
trap '_sigterm' SIGTERM

wait $pool_manager_pid
