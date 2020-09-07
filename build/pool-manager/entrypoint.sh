#!/bin/bash
#/*
#Copyright 2020 The OpenEBS Authors
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#	http://www.apache.org/licenses/LICENSE-2.0
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.
#*/

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
