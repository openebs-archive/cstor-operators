# Copyright © 2020 The OpenEBS Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/usr/bin/env bash

set -ex

echo "Install cstor-operators CRDs"

kubectl apply -f ./deploy/crds

echo "Apply RBAC rules"

kubectl apply -f ./deploy/rbac.yaml

echo "Install NDM-Operator"

kubectl apply -f ./deploy/ndm-operator.yaml

echo "Install cStor-Operators"

kubectl apply -f ./deploy/cstor-operator.yaml

echo "Install CSI"

kubectl apply -f ./deploy/csi-operator.yaml

sleep 5

echo "Verify CSI installation"

kubectl get pods -n kube-system -l role=openebs-csi

echo "Verify cstor-operators installation"

kubectl get pod -n openebs
