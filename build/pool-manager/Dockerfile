#
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
# This Dockerfile builds a recent cspi-mgmt using the latest binary from
# cspi-mgmt  releases.
#

#openebs/cstor-base is the image that contains cstor related binaries and
#libraries - zpool, zfs, zrepl
#FROM openebs/cstor-base:ci
ARG BASE_IMAGE
FROM $BASE_IMAGE

COPY pool-manager /usr/local/bin/
COPY entrypoint.sh /usr/local/bin/

RUN echo '#!/bin/bash\nif [ $# -lt 1 ]; then\n\techo "argument missing"\n\texit 1\nfi\neval "$*"\n' >> /usr/local/bin/execute.sh

RUN chmod +x /usr/local/bin/execute.sh
RUN apt install netcat -y
RUN chmod +x /usr/local/bin/entrypoint.sh

ARG ARCH
ARG DBUILD_DATE
ARG DBUILD_REPO_URL
ARG DBUILD_SITE_URL

LABEL org.label-schema.name="cstor-pool-manager"
LABEL org.label-schema.description="Pool manager for cStor pool"
LABEL org.label-schema.schema-version="1.0"
LABEL org.label-schema.build-date=$DBUILD_DATE
LABEL org.label-schema.vcs-url=$DBUILD_REPO_URL
LABEL org.label-schema.url=$DBUILD_SITE_URL

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
EXPOSE 7676
