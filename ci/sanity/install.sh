#!/usr/bin/env bash
echo "Install cstor-operators CRDs"

kubectl apply -f ./deploy/crds

echo "Apply RBAC rules"

kubectl apply -f ./deploy/rbac.yaml

echo "Install NDM-Operator"

kubectl apply -f ./deploy/ndm-operator.yaml

echo "Install cStor-Operators"

kubectl apply -f ./deploy/cstor-operator.yaml

echo "Install CSI"

kubectl apply -f ./ci/artifacts/csi-operator-ubuntu-18.04.yaml

sleep 5

echo "Verify CSI installation"

kubectl get pods -n kube-system -l role=openebs-csi

echo "Verify cstor-operators installation"

kubectl get pod -n openebs