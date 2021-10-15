# Volume Migration when the underlying cStor pool is lost

## Intro
- There can be situations where a node can be lost and disks attached to the node will also be lost if the disks are ephemeral. This is very common for users having Kubernetes autoscale feature enabled and nodes scale down and scale-up based on the demand.

**Note:** Meaning of lost node and lost disk:
Lost Node: When a node in the Kubernetes cluster does not come up again with the same name and hostname label.
Lost Disk(s): Ephemeral disk(s) attached to the lost node.

So a cStor pool is lost when one of the following situation occur:
- If node is lost.
- If one or more disks participating in the cStor pool are lost and the pool configuration is stripe.
- If the cStor pool configuration is mirror and all the disks participating in any raid group are lost.
- If the cStor pool configuration is raidz and if more than 1 disk in any raid group is lost.
- If the cStor pool configuration is raidz2 and if more than 2 disks in any raid group are lost.

If the volume replica that resided on the lost pool was configured in HA mode then the volume replica can be migrated to a new cStor pool.

**NOTE:** A volume is in HA if the volume has more than 2 replicas.

This document describes the steps to be followed for migrating a volume replica from a failed cStor pool to a new cStor pool. 

Consider an example where you have following nodes in a Kubernetes cluster with disks attached it:

    Worker-1 (bd-1(w1), bd-2(w1) are attached to worker1)

    Worker-2 (bd-1(w2), bd-2(w2) are attached to worker 2)

    Worker-3 (bd-1(w3), bd-2(w3) are attached to worker 3)

**NOTE**: Disks attached to a node are represented by a blockdevice(bd) in OpenEBS installed cluster. A block device is of the form `bd-<some-hash>`. For example, bd-1(w1), bd-3(w2) etc are BD resources. For illustration purpose hash is not included in BD name, e.g. `bd-1(w1)` represents a block device attached to worker-1.

## Reproduce the cStor pool lost situation?

#### In cloud environment
We can remove nodes from the Kubernetes cluster managed by an auto-scaler group(ASG) in cloud-managed Kubernetes services e.g. EKS and GCP. If the nodes are removed then the attached volumes/disks on the removed node will also get deleted if the attached volumes/disks are ephemeral.

#### On-Premise
On-Premise can have following situations:
- Node got corrupted but corrupted will never come back but disks still have the data intact(In this case follow steps mentioned [here](migrate_pool_by_migrating_disks.md).
- Disks attached to a node got corrupted but the node is operational.
- Node and disk both got corrupted together(Possible but highly unlikely).

## Infrastructure details
    Following are infrastructure details where reproduced the situation
    
    **Kubernetes Cluster**: Amazon(EKS)

    **Kubernetes Version**: 1.15.0

    **OpenEBS Version**: 1.12.0(Deployed OpenEBS by enabling **feature-gates="GPTBasedUUID"** since EBS volumes in EKS are virtual and some mechanisam is reuired to identify the disks uniquely).

    **Node OS**: Ubuntu 18.04

## Installation of cStor setup
Created CStor pools using CSPC API by following [doc](../quick.md) and then provisioned cStor-CSI volume with HA configuration(storage replica count as 3) on top of cStor pools.

### OpenEBS setup
List of resources that are created after provisioning cStor pools and volumes

- To know about cStorpoolcluster
```sh
$ Kubectl get cspc -n openebs
NAME         HEALTHYINSTANCES   PROVISIONEDINSTANCES   DESIREDINSTANCES   AGE
cstor-cspc   3                  3                      3                  108s
```
- To know more details about the cStor pool instances
```sh
$ kubectl get cspi -n openebs
NAME              HOSTNAME            ALLOCATED  FREE   CAPACITY  READONLY  PROVISIONEDREPLICAS  HEALTHYREPLICAS  TYPE    STATUS  AGE
cstor-cspc-4tr5   ip-192-168-52-185   98k        9630M  9630098k  false     1                    1                stripe  ONLINE  4m26s
cstor-cspc-xnxx   ip-192-168-79-76    101k       9630M  9630101k  false     1                    1                stripe  ONLINE  4m25s
cstor-cspc-zdvk   ip-192-168-29-217   98k        9630M  9630098k  false     1                    1                stripe  ONLINE  4m25s
```
- To know more about the cStor volumes resources
```sh
$ kubectl get cvc,cv,cvr -n openebs
NAME                                                                          STATUS   AGE
cstorvolumeconfig.cstor.openebs.io/pvc-81746e7a-a29d-423b-a048-76edab0b0826   Bound    7m3s

NAME                                                                                           USED   ALLOCATED   STATUS    AGE
cstorvolumereplica.cstor.openebs.io/pvc-81746e7a-a29d-423b-a048-76edab0b0826-cstor-cspc-4tr5   6K     6K          Healthy   7m3s
cstorvolumereplica.cstor.openebs.io/pvc-81746e7a-a29d-423b-a048-76edab0b0826-cstor-cspc-xnxx   6K     6K          Healthy   7m3s
cstorvolumereplica.cstor.openebs.io/pvc-81746e7a-a29d-423b-a048-76edab0b0826-cstor-cspc-zdvk   6K     6K          Healthy   7m3s

NAME                                                                    STATUS    AGE    CAPACITY
cstorvolume.cstor.openebs.io/pvc-81746e7a-a29d-423b-a048-76edab0b0826   Healthy   7m3s   5Gi
```

Following is the state of the system when one of the cStor pool is lost:

```sh
$ kubectl get cspc -n openebs
NAME         HEALTHYINSTANCES   PROVISIONEDINSTANCES   DESIREDINSTANCES   AGE
cstor-cspc   2                  3                      3                  20m
```
- Pool Manager details
```sh
$ kubectl get po -n openebs -l app=cstor-pool
NAME                               READY   STATUS    RESTARTS   AGE
cstor-cspc-4tr5-8455898c74-c7vbv   0/3     Pending   0          6m17s
cstor-cspc-xnxx-765bc8d899-7696q   3/3     Running   0          20m
cstor-cspc-zdvk-777df487c8-l62sv   3/3     Running   0          20m
```
In the above output cStor pool manager i.e **cstor-cspc-4tr5** which was scheduled on lost node is in pending state and output of CSPC also shows only two HealthyCSPIInstances.

## Steps to be followed to migrate the volume from failed cStor pool to a new cStor pool or an existing cStor pool of the same CSPC

**NOTE**: The CStorVolume related to the volume replicas that want to migrate should be **Healthy** then only we can perform the steps.

### Step1: Remove the cStorVolumeReplicas from the lost pool
This step is required to remove the pool from the lost node. Before removing the pool first we need to remove cStorVolumeReplicas in the pool or else the admission server will reject the scale down request of admission server. This can be achieved by removing the pool entry from the CStorVolumeConfig(CVC) spec section.

**Note**: This step will succeed only if the cstorvolume and target pod are in running state.

Edit and change the CVC resource corresponding to the volume
```sh
...
...
  policy:
   provision:
    replicaAffinity: false
     replica: {}
     replicaPoolInfo:
     - poolName: cstor-cspc-4tr5
     - poolName: cstor-cspc-xnxx
     - poolName: cstor-cspc-zdvk
...
...
```
To
```sh
$ kubectl edit cvc pvc-81746e7a-a29d-423b-a048-76edab0b0826 -n openebs
...
...
  policy:
   provision:
    replicaAffinity: false
     replica: {}
     replicaPoolInfo:
     - poolName: cstor-cspc-xnxx
     - poolName: cstor-cspc-zdvk
...
...
cstorvolumeconfig.cstor.openebs.io/pvc-81746e7a-a29d-423b-a048-76edab0b0826 edited
```
From the above spec **cstor-cspc-4tr5** CSPI entry is removed from CVC under spec. Repeat the same thing for all the volumes which have cStor volume replicas on the lost pool i.e cstor-cspc-4tr5. We can get list of volume replicas in lost pool using the following command

```sh
$ kubectl get cvr -n openebs -l cstorpoolinstance.openebs.io/name=cstor-cspc-4tr5
NAME                                                       USED   ALLOCATED   STATUS    AGE
pvc-81746e7a-a29d-423b-a048-76edab0b0826-cstor-cspc-bf9h   6K     6K          Healthy   4m7s
```

### Step2: Remove the finalizer from cStor volume replicas
This step is required to remove the `cstorvolumereplica.openebs.io/finalizer` finalizer from CVRs which were present on the lost cStor pool. After removing the finalizer CVR will be deleted from etcd. Usually, finalizer should be removed by pool-manager pod since the pod is not in running state manual intervention is required to remove the finalizer
```sh
$ kubectl get cvr -n openebs
NAME                                                       USED   ALLOCATED   STATUS    AGE
pvc-81746e7a-a29d-423b-a048-76edab0b0826-cstor-cspc-xnxx   6K     6K          Healthy   52m
pvc-81746e7a-a29d-423b-a048-76edab0b0826-cstor-cspc-zdvk   6K     6K          Healthy   52m
```

After this step, the scaledown of CStorVolume will be successful. One can verify from events on corresponding CVC
```sh
$ kubectl describe cvc <pv_name> -n openebs
Events:
Type     Reason                 Age    From                         Message
----     ------                 ----   ----                         -------
Normal   ScalingVolumeReplicas  6m10s  cstorvolumeclaim-controller  successfully scaled volume replicas to 2
```

### Step3: Remove the pool spec from CSPC belongs to lost node
Edit the CSPC spec using `kubectl edit cspc <cspc_name> -n openebs` and remove the pool spec belongings to nodes which no longer exist in the cluster. Once the spec was removed from the pool output then DesiredInstances will be 2. It can be verified using `kubectl get cspc -n openebs` will looks like
```sh
$ kubectl get cspc -n openebs
NAME         HEALTHYINSTANCES   PROVISIONEDINSTANCES   DESIREDINSTANCES   AGE
cstor-cspc   2                  3                      2                  56m
```
Since the pool manager pod is not in running state and because of the existence of pool protection finalizer i.e `openebs.io/pool-protection` on CSPI. CSPC-Operator was not able to delete the CSPI waiting for pool protection finalizer to get removed. Since the CSPI is not deleted, the ProvisionedInstances count is not updated. To fix this `openebs.io/pool-protection` finalizer should be removed from the cspi which was on the lost node.
```sh
kubectl edit cspi  cstor-cspc-4tr5
cstorpoolinstance.cstor.openebs.io/cstor-cspc-4tr5 edited
```

After removing finalizer Healthy, Provisioned and Desired instances will match as shown below
```sh
$ kubectl get cspc -n openebs
NAME         HEALTHYINSTANCES   PROVISIONEDINSTANCES   DESIREDINSTANCES   AGE
cstor-cspc   2                  2                      2                  68m
```
### Step4: Scale the cStorVolumeReplicas back to 3 using CVC
Scale the CStorVolumeRepilcas back to 3 to the new or existing cStor pool where a volume replica of the same volume doesn't exist. Since in the cluster there were no extra CStorPoolInstance so scaled the cStor pool by adding a new node pool spec. If you have already a spare CStorPoolInstance where you can place this volume replica, you do not need to scale the CStorPoolCluster(CSPC)

**NOTE:** A CStorVolume is a collection of 1 or more volume replicas and no two replicas of a CStorVolume should reside on the same CStorPoolInstacne. CStorVolume is a basically a custom resource and a logical aggregated representation of all the underlying cStor volume replicas for this particular volume.

```sh
$ kubectl get cspi -n openebs
NAME             HOSTNAME           ALLOCATED  FREE   CAPACITY  READONLY  PROVISIONEDREPLICAS  HEALTHYREPLICAS  TYPE    STATUS  AGE
cstor-cspc-bf9h  ip-192-168-49-174  230k       9630M  9630230k  false     0                    0                stripe  ONLINE  66s
```
Add above newly created CStorPoolInstance i.e cstor-cspc-bf9h under CVC.Spec
```sh
$ kubectl edit cvc pvc-81746e7a-a29d-423b-a048-76edab0b0826 -n openebs
...
...
spec:
 policy:
  provision:
   replicaAffinity: false
  replica: {}
  replicaPoolInfo:
  - poolName: cstor-cspc-bf9h
  - poolName: cstor-cspc-xnxx
  - poolName: cstor-cspc-zdvk
...
...
```
Repeat the same thing for all the scaled down cStor volumes and verify whether all the newly provisioned CStorVolumeReplica(CVR) are **Healthy**.
```sh
$ kubectl get cvr -n openebs
NAME                                                       USED   ALLOCATED   STATUS    AGE
pvc-81746e7a-a29d-423b-a048-76edab0b0826-cstor-cspc-bf9h   6K     6K          Healthy   11m
pvc-81746e7a-a29d-423b-a048-76edab0b0826-cstor-cspc-xnxx   6K     6K          Healthy   96m
pvc-81746e7a-a29d-423b-a048-76edab0b0826-cstor-cspc-zdvk   6K     6K          Healthy   96m
```

```sh
$ kubectl get cspi -n openebs
NAME              HOSTNAME            ALLOCATED  FREE   CAPACITY  READONLY  PROVISIONEDREPLICAS  HEALTHYREPLICAS  TYPE    STATUS  AGE
cstor-cspc-bf9h  ip-192-168-49-174    230k       9630M  9630230k  false     1                    1                stripe  ONLINE  66s
cstor-cspc-xnxx   ip-192-168-79-76    101k       9630M  9630101k  false     1                    1                stripe  ONLINE  4m25s
cstor-cspc-zdvk   ip-192-168-29-217   98k        9630M  9630098k  false     1                    1                stripe  ONLINE  4m25s
```
By comparing to the [previous](#openebs-setup) outputs of cStor pool has been migrated from lost node to new node.
