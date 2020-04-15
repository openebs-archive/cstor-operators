#!/bin/bash

# Downloads the most recently released ndm-operator yaml
# to ci/install_artifacts/ndm-operator

# Sets up ndm-operator kustomization yaml

NDM_RELEASE_TAG=v0.4.9
mkdir -p actions_ci/install_artifacts/ndm-operator
echo "Downloading ndm-operator yaml..."
wget -O actions_ci/install_artifacts/ndm-operator/ndm-operator.yaml https://raw.githubusercontent.com/openebs/node-disk-manager/$NDM_RELEASE_TAG/ndm-operator.yaml


cat <<EOF >actions_ci/install_artifacts/ndm-operator/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: openebs
images:
  - name: openebs/node-disk-manager-amd64
    newTag: $NDM_RELEASE_TAG
  - name: openebs/node-disk-operator-amd64
    newTag: $NDM_RELEASE_TAG
  - name: openebs/node-disk-exporter-amd64
    newTag: $NDM_RELEASE_TAG
resources:
- ndm-operator.yaml
EOF