#!/usr/bin/env bash

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

function IsPoolHealthy() {
MAX_RETRY=$1
for i in $(seq 1 $MAX_RETRY) ; do
poolStatus=$(kubectl get cspi -n openebs -o=jsonpath='{.items[*].status.phase}')
 if [ "$poolStatus" == "ONLINE" ]; then
  echo "CSPI is ONLINE!"
  break
 else
  echo "Waiting for CSPI to come online"
  kubectl get cspi -n openebs
 if [ "$i" == "$MAX_RETRY" ] && [ "$poolStatus" != "ONLINE" ]; then
  echo "CSPI did not come ONLINE"
  echo "Dumping CSPC-Operator logs"
  dumpCSPCOperatorLogs 100
  exit 1
 fi
 fi
 sleep 5
 done
}

function IsPVCBound(){
PVC_NAME=$1
PVC_MAX_RETRY=$2
for i in $(seq 1 $PVC_MAX_RETRY) ; do
 PVCStatus=$(kubectl get pvc $PVC_NAME --output="jsonpath={.status.phase}")
 if [ "$PVCStatus" == "Bound" ]; then
 echo "PVC $PVC_NAME bound successfully"
 break
 else
 echo "Waiting for $PVC_NAME to be bound"
 kubectl get pvc $PVC_NAME
 if [ "$i" == "$PVC_MAX_RETRY" ] && [ "$PVCStatus" != "Bound" ]; then
 echo "PVC $PVC_NAME NOT bound"
 echo "Describing PVC..."
 kubectl describe pvc $PVC_NAME
 echo "Listing pods in openebs namespace"
 kubectl get pod -n openebs
 echo "Listing pods in kube-system namespace"
 kubectl get pod -n kube-system
 exit 1
 fi
 fi
 sleep 5
done
}

function IsPodRunning(){
POD_NAME=$1
POD_MAX_RETRY=$2
for i in $(seq 1 $POD_MAX_RETRY) ; do
 PodStatus=$(kubectl get pod $POD_NAME --output="jsonpath={.status.phase}")
 if [ "$PodStatus" == "Running" ]; then
 echo "Pod $POD_NAME running"
 break
 else
 echo "Waiting for $POD_NAME to be in running"
 kubectl get pod $POD_NAME
 if [ "$i" == "$POD_MAX_RETRY" ] && [ "$PodStatus" != "Running" ]; then
 echo "Pod $POD_NAME NOT running!"
 echo "Describing Pod $POD_NAME ..."
 kubectl describe pod $POD_NAME
 exit 1
 fi
 fi
 sleep 5
done
}

function dumpCSPCOperatorLogs() {
  LC=$1
  CSPCPOD=$(kubectl get pods -o jsonpath='{.items[?(@.spec.containers[0].name=="cspc-operator")].metadata.name}' -n openebs)
  kubectl logs --tail=${LC} $CSPCPOD -n openebs
  printf "\n\n"
}

function getBD(){
nodeName=$1
BD_RETRY=$2
for i in $(seq 1 $BD_RETRY) ; do
 blockDeviceNames=$(kubectl get bd -n openebs -l kubernetes.io/hostname="$nodeName" -o=jsonpath='{.items[?(@.spec.details.deviceType=="sparse")].metadata.name}')
 if [ "$blockDeviceNames" != "" ]; then
 bdName=$(echo "$blockDeviceNames" | awk '{print $1}')
 echo "Got BD $bdName"
 break
 else
 echo "Waiting for a block device to come up"
 kubectl get bd -n openebs
 if [ "$i" == "$BD_RETRY" ] && [ "$bdName" == "" ]; then
 echo "No block devices found!"
 echo "Listing pod in openebs namespace ..."
 kubectl get pod -n openebs
 exit 1
 fi
 fi
 sleep 5
done

}

function wait_for_resource_deletion(){
    resource_kind=$1
    namespace=$2
    while true; do
        resource_list=$(kubectl get "$resource_kind" -n "$namespace" --no-headers)
        echo "$resource_list"
        if [ -z "$resource_list" ]; then
            break
        fi
        echo "Waiting for resource $resource_kind to get delete in namespace $namespace"
        sleep 5
    done
}

echo "Preparing the CSPC YAML from template"
nodeName=$(kubectl get node -o=jsonpath={.items[0].metadata.name})

getBD $nodeName 30

sed  "s/NODE_NAME/$nodeName/g;s/BD_NAME/$bdName/g" ./ci/artifacts/cspc-template.yaml > ./ci/sanity/cspc.yaml
echo "Following CSPC YAML will be applied"
echo "<-------------------------------------------------------------------------->"
cat ./ci/sanity/cspc.yaml
echo "<-------------------------------------------------------------------------->"
echo "Applying the prepared CSPC YAML"
kubectl apply -f ./ci/sanity/cspc.yaml

IsPoolHealthy 50

## Once the pool is healthy verify whether cachefile stored in persistent path
## Get the CSPC name
echo "Verifying whether pool cachefile is stored in persistent path"
cspcName=$(kubectl get cspc -n openebs -o=jsonpath={.items[0].metadata.name})
if [ $? -ne 0 ]; then
    echo "Failed to get CSPC name"
    exit 1
fi
## verify whether cache file present in persistent path
ls -lrth /var/openebs/cstor-pool/${cspcName}/pool.cache
if [ $? -ne 0 ]; then
    echo "cache file is not present in persistent path"
    exit 1
fi

echo "Applying cstor-csi storage class"
kubectl apply -f ./ci/artifacts/csi-storageclass.yaml
echo "Deploying Busy-Box pod to use the cStor CSI volume..."
kubectl apply -f ./ci/artifacts/busybox-csi-cstor-sparse.yaml

IsPVCBound csi-claim 30
IsPodRunning busybox 30

echo "Deleting BusyBox pod and check for the iscsi session cleaned properly..."
kubectl delete -f ./ci/artifacts/busybox-csi-cstor-sparse.yaml

kubectl wait --for=delete pod -l app=busybox --timeout=600s
sleep 2
sessionCount=$(sudo iscsiadm -m session | wc -l)
if [ $sessionCount -ne 0 ]; then
    echo "iSCSI session not cleaned up successfully --- $sessionCount"
    exit 1
fi

wait_for_resource_deletion pvc ""
wait_for_resource_deletion cvr openebs
wait_for_resource_deletion cv openebs

## Delete CSPC and wait till CSPC and all dependents gets deleted
## Delete CSPC
kubectl delete -f ./ci/sanity/cspc.yaml

wait_for_resource_deletion cspc openebs
wait_for_resource_deletion cspi openebs

## Running integration test
make integration-test
if [ $? -ne 0 ]; then
    echo "CStor integration test has failed"
    exit 1
fi
