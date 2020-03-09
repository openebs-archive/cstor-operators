#!/bin/sh

set -ex

rm /usr/local/bin/zrepl
exec /usr/local/bin/pool-manager start
exec service ssh start
exec service rsyslog start
