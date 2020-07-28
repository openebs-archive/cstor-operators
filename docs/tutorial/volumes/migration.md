
## How to Migrate Data Replica Across Nodes

This doc explains how to migrate data from one node to another without any down time of application.
For example here we are going to migrate Data replica from Node2 to Node3 in kubernetes cluster.

#### Create cStor Pools(CSPC):

Create cStor pools by following the steps mentioned here. Once the pools are created wait till all the cStor pools marked as Healthy. Check the cStor pools status by executing `kubectl get cspc -n openebs` command(cspc - cStorPoolCluster)

```sh
NAME                 HEALTHYINSTANCES   PROVISIONEDINSTANCES   DESIREDINSTANCES   AGE
cspc-stripe-pool     3                  3                      3                  4m13s
```

To know more information about pools information following command will helpful `kubectl get cspi -n openebs`(cStorPoolInstances)

```sh
NAME                    HOSTNAME        ALLOCATED    FREE    CAPACITY  READONLY   TYPE     STATUS   AGE
cspc-stripe-pool-6qkw   e2e1-node1      81500        9630M   10G       false      stripe   ONLINE   21m
cspc-stripe-pool-pn9p   e2e1-node3      84500        9630M   10G       false      stripe   ONLINE   21m
cspc-stripe-pool-psz5   e2e1-node2      56k          9630M   10G       false      stripe   ONLINE   21m
```
#### Create CStor CSI Volume:

Create CSI volumes on cStor pools created above by following the steps mentioned here. As part of volume provisioning a resource called `CStorVolumeConfig` will created, and once the volume provisioned successfully then CVC(cStorVolumeConfig) status will updated to Bound which means all the CStorVolume resources are successfully created. Following is the command which will help to get CVC status `kubectl get cvc -n openebs`

```sh
kubectl get cvc -n openebs
NAME                                       STATUS     AGE
pvc-d1b26676-5035-4e5b-b564-68869b023306   Bound      5m56s
```

Get CVC resource to get node specific details where volume replicas are exists, using the command `kubectl get cvc <PV_NAME> -n <openebs_namespace> -o yaml`
Based on given 2 replicaCount in StorageClass there will be 2 CStorvolume replicas will be created and distributed across the 2 different cstorpool running instances in different nodes.

```sh
apiVersion: cstor.openebs.io/v1
kind: CStorVolumeConfig
name: pvc-d1b26676-5035-4e5b-b564-68869b023306
…
…
spec:
  capacity:
    storage: 5Gi
…
...
  policy:
    replicaPoolInfo:
    - poolName: cspc-stripe-pool-6qkw
    - poolName: cspc-stripe-pool-pn9p
status:
  phase: Bound
  poolInfo:
  - cspc-stripe-pool-6qkw
  - cspc-stripe-pool-pn9p
```
From the above output `status.poolInfo` shows, CStorVolumeReplicas(CVR) are created on cStor pools `cspc-stripe-pool-6qkw` and `cspc-stripe-pool-pn9p` scheduled in node `e2e1-node1` and `e2e1-node2` respectively.
Info under spec i.e spec.policy.replicaPoolInfo can be changed as request scale cStorVolumeReplicas operations.


To know more details of CVR we can get from `kubectl get cvr -n openebs`
```sh
NAME                                                             USED     ALLOCATED      STATUS    AGE
pvc-d1b26676-5035-4e5b-b564-68869b023306-cspc-stripe-pool-6qkw   1.47G    1.26G          Healthy   15h
pvc-d1b26676-5035-4e5b-b564-68869b023306-cspc-stripe-pool-pn9p   1.47G    1.26G          Healthy   15h
```

### Steps to migrate data from one node to other:

There are 2 steps required for this operation:

- Scale the CStorVolumeReplica to the desired node.
- Scale down the CStorVolumeReplicas from unwanted node.

### **Note**: Scale down of CStorVolumeReplicas will be allowed only if corresponding CStorVolume is in Healthy state.

#### 1: Scale the CStorVolumeReplicas to Desired Node

Firstly we have to get cStor pool name which doesn’t have corresponding volume CVR created on it.
We can get pool name name which doesn’t have CVR in it by comparing outputs of 
`kubectl get cspi -n openebs -l openebs.io/cstor-pool-cluster=<cspc_name>` and 
`kubectl get -n openebs cvc <pv_name> -o yaml` as explained above.

In this example, CVR of volume `pvc-d1b26676-5035-4e5b-b564-68869b023306` doesn’t not exist in cStor pool `cspc-stripe-pool-psz5`, so we can edit the CVC and add the pool name list under `policy.replicaPoolInfo`.

```sh
$ kubectl -n openebs cvc pvc-d1b26676-5035-4e5b-b564-68869b023306

```

```sh
apiVersion: cstor.openebs.io/v1
kind: CStorVolumeConfig
name: pvc-d1b26676-5035-4e5b-b564-68869b023306
…

Once the pool was added into the `spec.replicaPoolInfo` than the status of CVC will be updated with a new pool name as shown below, and raise an events which eventually creates new CVR on the newly added pool. We can get the CVR status by executing `kubectl get cvr -n openebs`

…
spec:
…
...
  policy:
    replicaPoolInfo:
    - poolName: cspc-stripe-pool-6qkw
    - poolName: cspc-stripe-pool-pn9p
    - poolName: cspc-stripe-pool-psz5
…
…
status:
  poolInfo:
  - cspc-stripe-pool-6qkw
  - cspc-stripe-pool-pn9p
  - cspc-stripe-pool-psz5
```
Events: Events on corresponding CVC
```sh
Events:
  Type        Reason                            Age                      From                                     Message
  ----           ------                                 ----                        ----                                         -------
  Normal    ScalingVolumeReplicas  14s (x2 over 15h)  cstorvolumeclaim-controller  successfully scaled volume replicas to 3
```

CVR status(by executing command):
```sh
$ kubectl get cvr -n openebs
NAME                                                             USED     ALLOCATED  STATUS                     AGE
pvc-d1b26676-5035-4e5b-b564-68869b023306-cspc-stripe-pool-6qkw   1.48G    1.25G      Healthy                    16h
pvc-d1b26676-5035-4e5b-b564-68869b023306-cspc-stripe-pool-pn9p   1.48G    1.26G      Healthy                    16h
pvc-d1b26676-5035-4e5b-b564-68869b023306-cspc-stripe-pool-psz5   91.4M    42.4M      ReconstructingNewReplica   33s
```
	
In the above output newly created CVRs convey that it was reconstructing data from scratch by talking to peer replicas. Wait till the newly created CVR marked as Healthy. To know status periodically execute `kubectl get cvr -n openebs` command

```sh
NAME                                                             USED      ALLOCATED   STATUS     AGE
pvc-d1b26676-5035-4e5b-b564-68869b023306-cspc-stripe-pool-6qkw   1.48G     1.25G       Healthy    16h
pvc-d1b26676-5035-4e5b-b564-68869b023306-cspc-stripe-pool-pn9p   1.48G     1.25G       Healthy    16h
pvc-d1b26676-5035-4e5b-b564-68869b023306-cspc-stripe-pool-psz5   1.48G     1.25G       Healthy    5m28s
```

Note:
Reconstructing will take time depending on the amount of data requires to rebuild.

#### 2: Scale down the CStorVolumeReplicas from unwanted nodes

Once the newly created CVR is marked as Healthy then we can remove the unwanted pool name from Spec of CVC and save it. In this example we need to remove the data from the pool `cspc-stripe-pool-pn9p` which was scheduled on Node `e2e1-node2`. Once the pool name is removed from CVC `spec.policy.replicaPoolInfo` then corresponding CVR in that pool will be deleted. We can describe CVC to know the generated operation events and status of CVC updated with latest scale changes.

Events on CVR:
```sh
Events:
  Type       Reason                            Age                       From                                      Message
  ----          ------                                 ----                         ----                                         -------
  Warning  ScalingVolumeReplicas  4s (x2 over 64m)   cstorvolumeclaim-controller  Scaling down volume replicas from 3 to 2 is in progress
  Normal   ScalingVolumeReplicas  4s (x2 over 64m)   cstorvolumeclaim-controller  successfully scaled volume replicas to 2
```

From output of `kubectl get cspi -n openebs`
```sh
NAME                    HOSTNAME     ALLOCATED  FREE    CAPACITY   READONLY   TYPE     STATUS   AGE
cspc-stripe-pool-6qkw   e2e1-node1   1260M      8370M   9630M       false     stripe   ONLINE   17h
cspc-stripe-pool-pn9p   e2e1-node2   5040k      9620M   9625040k    false     stripe   ONLINE   16h
cspc-stripe-pool-psz5   e2e1-node3   1260M      8370M   9630M       false     stripe   ONLINE   17h
```

Awesome!!! from above storage usage I am able to successfully migrate the data from one node to other without any down time of application.
