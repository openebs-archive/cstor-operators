# Pool Migration when underlying disks were migrated to different node

## Intro
- There can be situations where your Kubernetes cluster scales down and scales up again(to save cost). This can cause nodes to come with a different hostName and node name than the previous existing nodes, meaning the nodes that have come up are new nodes and not the same as previous nodes that earlier existed.
- When the new nodes come up, the disks that were attached to the older nodes now get attached to the newer nodes.

Consider an example where you have following nodes in a Kubernetes cluster with disks attached it:
 
    Worker-1 (bd-1(w1), bd-2(w1) are attached to worker1)
  
    Worker-2 (bd-1(w2), bd-2(w2) are attached to worker 2)
  
    Worker-3 (bd-1(w3), bd-2(w3) are attached to worker 3)

**NOTE**: Disks attached to a node are represented by a blockdevice(bd) in OpenEBS installed cluster. A block device is of the form `bd-<some-hash>`.
      For example, bd-1(w1), bd-3(w2) etc are BD resources. For illustration purpose hash is not included in BD name, e.g. `bd-1(w1)` represents a block device attached to worker-1.

## What happens if Node Replacement Occurs?
  If node replacement occurs in your Kubernetes cluster then cStor pool manager pods will be in pending state and the pools and volumes will go offline. Workloads using those cStor volumes will not be able to perform read and write operations on the volume.

## How can this be fixed?
  We can perform a few manual [steps](#steps-to-bring-cstor-pool-back-online) to recover from this situation. But before we do this, the tutorial will illustrate a Node Replacement situation. So essentially we are trying to do the following:

**__Migrate CStorPool when nodes where replaced with new nodes but same disks were reattached to the new nodes__**

## Reproduce the Node Replacement situation?

#### In cloud environment
  Able to replace the nodes by deleting the node from the K8s cluster managed by autoscale groups(ASG). Experimented in EKS, GCP managed kubernetes cluster.

#### On-Premise
  Detach the disk from the node where the pool is running and attach to the different node then corresponding CSPC pool managers will get restarted from every 5 minutes due to livenessfailure(migrating disk to different node might be case where resources got exhausted on the node and pods were evicted and not able to schedule back on the node).

## Infrastructure details
  Following are infrastructure details where node replacements are performed

    **Kubernetes Cluster**: Amazon(EKS)

    **Kubernetes Version**: 1.15.0

    **OpenEBS Version**: 1.12.0(Deployed OpenEBS by enabling **feature-gates="GPTBasedUUID"** since EBS volumes in EKS are virtual and some mechanisam is reuired to identify the disks uniquely).
    
    **Node OS**: Ubuntu 18.04

## Installation of cStor setup
  Created CStor pools using CSPC API by following [doc](../quick.md) and then created cStor-CSI volume on top of cStor pools. 

### OpenEBS setup

  After installing control plane components on 3 node kubernetes cluster with one disk attached to each node
```sh
$ kubectl get nodes --show-labels
NAME                                           STATUS   ROLES    AGE    VERSION   LABELS
ip-192-168-55-201.us-east-2.compute.internal   Ready    <none>   127m   v1.15.9   kubernetes.io/hostname=ip-192-168-55-201
ip-192-168-8-238.us-east-2.compute.internal    Ready    <none>   127m   v1.15.9   kubernetes.io/hostname=ip-192-168-8-238
ip-192-168-95-129.us-east-2.compute.internal   Ready    <none>   127m   v1.15.9   kubernetes.io/hostname=ip-192-168-95-129
```

  Disks attached to the nodes were represented as
```sh
$ kubectl get bd -n openebs
NAME                                           NODENAME                                       SIZE          CLAIMSTATE   STATUS   AGE
blockdevice-4505d9d5f045b05995a5654b5493f8e0   ip-192-168-55-201.us-east-2.compute.internal   10737418240   Claimed      Active   50m
blockdevice-798dbaf214f355ada15d097d87da248c   ip-192-168-8-238.us-east-2.compute.internal    10737418240   Claimed      Active   50m
blockdevice-c783e51a80bc51065402e5473c52d185   ip-192-168-95-129.us-east-2.compute.internal   10737418240   Claimed      Active   50m
```

### CSPC used to create cStor pools

- Following is the CSPC API used to provision cStor pool on existing blockdevices
```yaml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cstor-cspc
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "ip-192-168-8-238"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-798dbaf214f355ada15d097d87da248c"
      poolConfig:
        dataRaidGroupType: "stripe"
    - nodeSelector:
        kubernetes.io/hostname: "ip-192-168-55-201"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-4505d9d5f045b05995a5654b5493f8e0"
      poolConfig:
        dataRaidGroupType: "stripe"
    - nodeSelector:
        kubernetes.io/hostname: "ip-192-168-95-129"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-c783e51a80bc51065402e5473c52d185"
      poolConfig:
        dataRaidGroupType: "stripe"
```
- After applying CSPC API to know **Healthy and Provisioned** pool count execute following command
```sh
$ kubectl get cspc -n openebs
NAME         HEALTHYINSTANCES   PROVISIONEDINSTANCES   DESIREDINSTANCES   AGE
cstor-cspc   3                  3                      3                  29m
```
- To know more details about the pools(CSPIs) like on which nodes cStor pools were scheduled, various capacity details of pool, Provisioned and Healthy volumereplica count can be known by executing following command
```sh
$ kubectl get cspi -n openebs
NAME              HOSTNAME            ALLOCATED   FREE   CAPACITY  READONLY  PROVISIONEDREPLICAS  HEALTHYREPLICAS  TYPE     STATUS   AGE
cstor-cspc-chjg   ip-192-168-95-129   143k        9630M  9630143k  false     1                    1                stripe   ONLINE   35m
cstor-cspc-h99x   ip-192-168-8-238    1420k       9630M  9631420k  false     1                    1                stripe   ONLINE   35m
cstor-cspc-xs4b   ip-192-168-55-201   154k        9630M  9630154k  false     1                    1                stripe   ONLINE   35m
```
- After inferring the above output cStor pools were created on following nodes and blockdevices
```
|----------------------------------------------------------------------------------------------------------------------|
| CSPI Name(pool name)|  Node Name                                    | BlockDevice Name                               |
|----------------------------------------------------------------------------------------------------------------------|
| cstor-cspc-chjg     |  ip-192-168-95-129.us-east-2.compute.internal | blockdevice-c783e51a80bc51065402e5473c52d185   |
| cstor-cspc-h99x     |  ip-192-168-8-238.us-east-2.compute.internal  | blockdevice-798dbaf214f355ada15d097d87da248c   |
| cstor-cspc-xs4b     |  ip-192-168-55-201.us-east-2.compute.internal | blockdevice-4505d9d5f045b05995a5654b5493f8e0   |
|----------------------------------------------------------------------------------------------------------------------|
```
### Replaced Nodes
  Performed node replacement action as specified [here](#reproduce-the-node-replacement-situation). Now the existing nodes were replaced with new nodes but the same disks were attached to the nodes(able to confirm from below outputs). Below are updated node details
```sh
$ kubectl get nodes --show-labels
NAME                                           STATUS   ROLES    AGE     VERSION  LABELS
ip-192-168-25-235.us-east-2.compute.internal   Ready    <none>   6m28s   v1.15.9  kubernetes.io/hostname=ip-192-168-25-235
ip-192-168-33-15.us-east-2.compute.internal    Ready    <none>   8m3s    v1.15.9  kubernetes.io/hostname=ip-192-168-33-15
ip-192-168-75-156.us-east-2.compute.internal   Ready    <none>   4m30s   v1.15.9  kubernetes.io/hostname=ip-192-168-75-156
```

Blockdevice names were not changed but node name and host names are updated with new node and hostname details
```sh
$ kubectl get bd -n openebs
NAME                                           NODENAME                                       SIZE          CLAIMSTATE   STATUS   AGE
blockdevice-4505d9d5f045b05995a5654b5493f8e0   ip-192-168-33-15.us-east-2.compute.internal    10737418240   Claimed      Active   3h8m
blockdevice-798dbaf214f355ada15d097d87da248c   ip-192-168-25-235.us-east-2.compute.internal   10737418240   Claimed      Active   3h8m
blockdevice-c783e51a80bc51065402e5473c52d185   ip-192-168-75-156.us-east-2.compute.internal   10737418240   Claimed      Active   3h8m
```
Once the nodes where replaced, the pool manager pods will remain in pending state due to existence of nodeselector on pool manager deployments. We can verify pool details using following command
```sh
$ kubectl get cspc -n openebs
NAME         HEALTHYINSTANCES   PROVISIONEDINSTANCES   DESIREDINSTANCES   AGE
cstor-cspc   0                  3                      3                  170m
```
For verifying pool manager pods
```
$ kubectl get po -n openebs -l app=cstor-pool
NAME                               READY   STATUS    RESTARTS   AGE
cstor-cspc-chjg-85f65ff79d-pq9d2   0/3     Pending   0          16m
cstor-cspc-h99x-57888d4b5-kh42k    0/3     Pending   0          15m
cstor-cspc-xs4b-85dbbbb59b-wvhmr   0/3     Pending   0          18m
```
In the above output **HEALTHYINSTANCES** were **0** and all the pool pods were in pending state because nodes were replaced with new nodes and still CSPC spec and cStor pool manager is pointing to old nodes.

**NOTE**: Please make sure that PROVISIONEDINSTANCES and DESIREDINSTANCES count should match before performing steps.

## Steps to bring cStor pool back online

### Step1: Update validatingwebhookconfiguration resource failurePolicy

This step is required to inform the kube-APIServer to ignore the error if kube-APIServer is not able to reach cStor admission server.
```sh
$ kubectl get validatingwebhookconfiguration openebs-cstor-validation-webhook 
kind: ValidatingWebhookConfiguration
metadata:
  name: openebs-cstor-validation-webhook
  ...
  ...
webhooks:
- admissionReviewVersions:
  - v1beta1
failurePolicy: Fail
  name: admission-webhook.cstor.openebs.io
...
...
```
In the above configuration update the **failurePolicy** from **Fail** to **Ignore**. Using kubectl edit command
```sh
$ kubectl edit validatingwebhookconfiguration openebs-cstor-validation-webhook
```

### Step2: Scaledown the admission

This step is required to skip the validations performed by cStor admission server when CSPC spec is updated with new node details.
```sh
$ kubectl scale deploy openebs-cstor-admission-server -n openebs --replicas=0
deployment.extensions/openebs-cstor-admission-server scaled
```

### Step3: Update the CSPC spec nodeSelector

This step is required to update the nodeSelector to point to new nodes instead of old nodeSelectors.
```sh
$ cat multiple_cspc.yaml 
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cstor-cspc
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "ip-192-168-25-235"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-798dbaf214f355ada15d097d87da248c"
      poolConfig:
        dataRaidGroupType: "stripe"
    - nodeSelector:
        kubernetes.io/hostname: "ip-192-168-33-15"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-4505d9d5f045b05995a5654b5493f8e0"
      poolConfig:
        dataRaidGroupType: "stripe"
    - nodeSelector:
        kubernetes.io/hostname: "ip-192-168-75-156"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-c783e51a80bc51065402e5473c52d185"
      poolConfig:
        dataRaidGroupType: "stripe"
```
Compare the above output to [previous](#cspc-used-to-create-cstor-pools) cspc output. NodeSelector have been updated with new node details i.e **kubernetes.io/hostname** values in poolSpec were updated. NodeSelector should be updated accordingly where the blockevices were attached. Apply the above configuration using
```sh
$ kubectl apply -f cspc.yaml
```

### Step4: Update the cspi spec nodeSelectors, labels and NodeName

This step is required to update the CSPI with correct node details by looking at the node where corresponding CSPI blockdevices were attached.
```sh
apiVersion: cstor.openebs.io/v1
kind: CStorPoolInstance
metadata:
  name: cstor-cspc-chjg
  namespace: openebs
  labels:
    kubernetes.io/hostname: ip-192-168-95-129
  ...
  ...
spec:
  dataRaidGroups:
  - blockDevices:
    - blockDeviceName: blockdevice-c783e51a80bc51065402e5473c52d185
  hostName: ip-192-168-95-129
  nodeSelector:
    kubernetes.io/hostname: ip-192-168-95-129
  poolConfig:
     ...
     ...
```
Get the node details on which the **blockdevice-c783e51a80bc51065402e5473c52d185** was attached and after fetching node details update hostName, nodeSelector values and kubernetes.io/hostname values in lables of CSPI with new details.
```sh
apiVersion: cstor.openebs.io/v1
kind: CStorPoolInstance
metadata:
  name: cstor-cspc-chjg
  namespace: openebs
 labels:
    kubernetes.io/hostname: ip-192-168-75-156
  ...
  ...
spec:
  dataRaidGroups:
  - blockDevices:
    - blockDeviceName: blockdevice-c783e51a80bc51065402e5473c52d185
  hostName: ip-192-168-75-156
  nodeSelector:
    kubernetes.io/hostname: ip-192-168-75-156
  poolConfig:
     ...
     ...
```
Update using `kubectl edit cspi <cspi_name> -n openebs`

**NOTE**: Repeat the same process for all other CSPIs which are in pending state and belongs to update CSPC.

### Step5: Verification

This step is identify whether pools has been imported or not. Updated CSPI will get events saying pool is successfully imported.
```sh
$ kubectl describe cspi cstor-cspc-xs4b -n openebs
...
...
Events:
  Type    Reason         Age    From               Message
  ----    ------         ----   ----               -------
  Normal  Pool Imported  2m48s  CStorPoolInstance  Pool Import successful: cstor-07c4bfd1-aa1a-4346-8c38-f81d33070ab7
```

### Step6: Scaleup the cStor admission server and update the validatingwebhookconfiguration

This step is required to bring back the cStor admission server into Running state. As well as admssion server is required to validate the modifications made to CSPC API in future.
```sh
$ kubectl scale deploy openebs-cstor-admission-server -n openebs --replicas=1
deployment.extensions/openebs-cstor-admission-server scaled
```

Update **failurePolicy to Fail** in validatingwebhookconfiguration.
```sh
$ kubectl edit validatingwebhookconfiguration openebs-cstor-validation-webhook
validatingwebhookconfiguration.admissionregistration.k8s.io/openebs-cstor-validation-webhook edited
```

## NOTE: To track scenario in automated way please refer to [this](https://github.com/openebs/cstor-operators/issues/100) issue.
