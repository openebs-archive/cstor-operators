#!/bin/sh

set -ex

CVC_API_SERVER_NETWORK="eth0"

CONTAINER_IP_ADDR=$(ip -4 addr show scope global dev "${CVC_API_SERVER_NETWORK}" | grep inet | awk '{print $2}' | cut -d / -f 1)

exec /usr/local/bin/cvc-operator --bind="${CONTAINER_IP_ADDR}" 1>&2
