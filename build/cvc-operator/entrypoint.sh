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
#!/bin/sh

set -ex

CVC_API_SERVER_NETWORK="eth0"

CONTAINER_IP_ADDR=$(ip -4 addr show scope global dev "${CVC_API_SERVER_NETWORK}" | grep inet | awk '{print $2}' | cut -d / -f 1)

exec /usr/local/bin/cvc-operator --bind="${CONTAINER_IP_ADDR}" 1>&2
