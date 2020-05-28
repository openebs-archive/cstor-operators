FROM ubuntu:18.04

RUN apt-get update && apt-get install -y \
    iproute2

ADD webhook /usr/local/bin/webhook



ARG ARCH
ARG DBUILD_DATE
ARG DBUILD_REPO_URL
ARG DBUILD_SITE_URL

LABEL org.label-schema.name="cstor-webhook"
LABEL org.label-schema.description="Webhook admission server for cStor"
LABEL org.label-schema.schema-version="1.0"
LABEL org.label-schema.build-date=$DBUILD_DATE
LABEL org.label-schema.vcs-url=$DBUILD_REPO_URL
LABEL org.label-schema.url=$DBUILD_SITE_URL

ENTRYPOINT ["/usr/local/bin/webhook"]

