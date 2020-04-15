#!/bin/bash

# Fetches cstor-operators YAML

# NOTE:
# This script copies the entire cstor-operators artifact from deploy directory

# Sets up cstor-operators kustomization yaml

IMAGE_TAG=$1
mkdir -p actions_ci/install_artifacts/cstor-operators
echo "Copying the cstor-operators artifacts"
cp -r deploy/* actions_ci/install_artifacts/cstor-operators

echo "Creating kustomization config for cstor-operators"
cat <<EOF >actions_ci/install_artifacts/cstor-operators/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: openebs
images:
  - name: openebs/cspc-operator
    newTag: $IMAGE_TAG
  - name: openebs/cvc-operator
    newTag: $IMAGE_TAG
  - name: openebs/cstor-webhook
    newTag: $IMAGE_TAG
resources:
- cstor-operator.yaml
EOF




