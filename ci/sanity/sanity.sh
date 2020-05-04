#!/usr/bin/env bash

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
 if [ "$PVCStatus" == "Running" ]; then
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
 bdName=$(kubectl get bd -n openebs -l kubernetes.io/hostname=$nodeName -o=jsonpath={.items[0].metadata.name})
 if [ "$bdName" != "" ]; then
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

IsPoolHealthy 30

echo "Applying cstor-csi storage class"
kubectl apply -f ./ci/artifacts/csi-storageclass.yaml
echo "Deploying Busy-Box pod to use the cStor CSI volume..."
kubectl apply -f ./ci/artifacts/busybox-csi-cstor-sparse.yaml

IsPVCBound csi-claim 30
IsPodRunning busybox 30