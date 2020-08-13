#
# This Dockerfile builds a recent cstor-volume-mgmt using the latest binary from
# cstor-volume-mgmt  releases.
#

FROM golang:1.13.6 as build

ARG TARGETPLATFORM

ENV GO111MODULE=on \
  CGO_ENABLED=1 \
  DEBIAN_FRONTEND=noninteractive \
  PATH="/root/go/bin:${PATH}"

WORKDIR /go/src/github.com/openebs/cstor-operator/

RUN apt-get update && apt-get install -y make git zip

COPY . .

RUN export GOOS=$(echo ${TARGETPLATFORM} | cut -d / -f1) && \
  export GOARCH=$(echo ${TARGETPLATFORM} | cut -d / -f2) && \
  GOARM=$(echo ${TARGETPLATFORM} | cut -d / -f3 | cut -c2-) && \
  make buildx.volume-manager

FROM ubuntu:16.04

ARG ARCH
ARG DBUILD_DATE
ARG DBUILD_REPO_URL
ARG DBUILD_SITE_URL

LABEL org.label-schema.name="cstor-volume-manager"
LABEL org.label-schema.description="Volume manager for cStor volumes"
LABEL org.label-schema.schema-version="1.0"
LABEL org.label-schema.build-date=$BUILD_DATE
LABEL org.label-schema.build-date=$DBUILD_DATE
LABEL org.label-schema.vcs-url=$DBUILD_REPO_URL
LABEL org.label-schema.url=$DBUILD_SITE_URL

RUN apt-get update; exit 0
RUN apt-get -y install rsyslog

RUN mkdir -p /usr/local/etc/istgt

COPY --from=build /go/src/github.com/openebs/cstor-operator/bin/volume-manager /usr/local/bin/
COPY --from=build /go/src/github.com/openebs/cstor-operator/build/volume-manager/entrypoint.sh /usr/local/bin/

RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT entrypoint.sh
EXPOSE 7676 7777
