## CStor Volume Policies:

CStor Volumes can be provisioned based on different policy configurations. CStorVolumePolicy has to be created prior to StorageClass 
and we have to mention the `CStorVolumePolicy` name in StorageClass parameters to provision cStor volume based on configured policy.

Following are list of policies that can be configured based on the requirements.

- [Replica Affinity to create a volume replica on specific pool](#replica-affinity)
- [Volume Target Pod Affinity](#volume-target-pod-affinity)
- [Volume Tunable](#volume-tunable)
- [Memory and CPU Resources QOS](#resource-request-and-limits) 
- [Toleration for target pod to ensure scheduling of target pods on tainted nodes](#target-pod-toleration)
- [NodeSelector for target pod to ensure scheduling of target pod on specific set of nodes](#target-pod-nodeselector)
- [Priority class for volume target deployment](#priority-class)

Below StorageClass example contains `cstorVolumePolicy` parameter having `csi-volume-policy` name set to configured the custom policy.

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: cstor-sparse-auto
provisioner: cstor.csi.openebs.io
allowVolumeExpansion: true
parameters:
  replicaCount: "1"
  cstorPoolCluster: "cspc-disk"
  cas-type: "cstor"
  fsType: "xfs"                 // default type is ext4
  cstorVolumePolicy: "csi-volume-policy"

```

If the volume policy is not created before volume provisioning and later want to change any of the policy it can be change
by editing the CStorVolumeConfig(CVC) resource as per volume bases which will be reconciled by the CVC controller
to the respected volume resources.

Each PVC create request will create a CStorVolumeConfig(cvc) resource which can be used to manage volume, its policies and any supported
day-2 operations (ex: scale up/down), per volume bases.

```bash
kubectl edit cvc <pv-name> -n openebs
```

```yaml
apiVersion: cstor.openebs.io/v1
kind: CStorVolumeConfig
metadata:
  annotations:
    openebs.io/persistent-volume-claim: "cstor-vol1"
    openebs.io/volume-policy: csi-volume-policy
    openebs.io/volumeID: pvc-25e79ecb-8357-49d4-83c2-2e63ebd66278
  creationTimestamp: "2020-07-22T11:36:13Z"
  finalizers:
  - cvc.openebs.io/finalizer
  generation: 3
  labels:
    cstor.openebs.io/template-hash: "3278395555"
    openebs.io/cstor-pool-cluster: cspc-sparse
  name: pvc-25e79ecb-8357-49d4-83c2-2e63ebd66278
  namespace: openebs
  resourceVersion: "1283"
  selfLink: /apis/cstor.openebs.io/v1/namespaces/openebs/cstorvolumeconfigs/pvc-25e79ecb-8357-49d4-83c2-2e63ebd66278
  uid: 389320d8-5f0b-439d-8ef2-59f4d01b393a
publish:
  nodeId: 127.0.0.1
spec:
  capacity:
    storage: 1Gi
  cstorVolumeRef:
    apiVersion: cstor.openebs.io/v1
    kind: CStorVolume
    name: pvc-25e79ecb-8357-49d4-83c2-2e63ebd66278
    namespace: openebs
    resourceVersion: "1260"
    uid: ea6e09f2-1e65-41ab-820a-ed1ecd14873c
  policy:
    provision:
      replicaAffinity: true
    replica:
      zvolWorkers: "1"
    replicaPoolInfo:
    - poolName: cspc-sparse-lh7n
    target:
      affinity:
        requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchExpressions:
            - key: openebs.io/target-affinity
              operator: In
              values:
              - percona
          namespaces:
          - default
          topologyKey: kubernetes.io/hostname
      auxResources:
        limits:
          cpu: 500m
          memory: 128Mi
        requests:
          cpu: 250m
          memory: 64Mi
      luWorkers: 8
      priorityClassName: system-cluster-critical
      queueDepth: "16"
      resources:
        limits:
          cpu: 500m
          memory: 128Mi
        requests:
        .
        .
        .
```
### Replica Affinity:

For StatefulSet applications, to distribute single replica volume on specific cstor pool we can use the `replicaAffinity` enabled scheduling.
This feature should be used with delay volume binding i.e. `volumeBindingMode: WaitForFirstConsumer` in StorageClass as shown below.

If `WaitForFirstConsumer` volumeBindingMode is set, then the csi-provisioner will wait for the scheduler to pick a node. The topology of
that selected node will then be set as the first entry in preferred list and will be used  by the volume controller to create the volume
replica on the cstor pool scheduled on preferred Node. 

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: cstor-sparse-auto
provisioner: cstor.csi.openebs.io
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
parameters:
  replicaCount: "1"
  cstorPoolCluster: "cspc-disk"
  cas-type: "cstor"
  cstorVolumePolicy: "csi-volume-policy"      // policy created with replicaAffinity set to true
```

This requires to be enabled via volume policy before provisioning the volume

```yaml
apiVersion: cstor.openebs.io/v1
kind: CStorVolumePolicy
metadata:
  name: csi-volume-policy
  namespace: openebs
spec:
  provision:
    replicaAffinity: true
```

### Volume Target Pod Affinity:

The Stateful workloads access the OpenEBS storage volume by connecting to the Volume Target Pod. 
Target Pod Affinity policy can be used to co-locate volume target pod on the same node as workload.
This feature makes use of the Kubernetes Pod Affinity feature that is dependent on the Pod labels. 
User will need to add the following label to both Application and volume Policy.

Configured Policy having target-affinity label for example, using `kubernetes.io/hostname` as a topologyKey in CStorVolumePolicy:

```yaml
apiVersion: cstor.openebs.io/v1
kind: CStorVolumePolicy
metadata:
  name: csi-volume-policy
  namespace: openebs
spec:
  target:
    affinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: openebs.io/target-affinity
            operator: In
            values:
            - fio-cstor                              // application-unique-label
        topologyKey: kubernetes.io/hostname
        namespaces: ["default"]                      // application namespace
```


Set the label configured in volume policy created above `openebs.io/target-affinity: fio-cstor` on the app pod which will be used to find pods, by label, within the domain defined by topologyKey.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: fio-cstor
  namespace: default
  labels:
    name: fio-cstor
    openebs.io/target-affinity: fio-cstor
```


### Volume Tunable:

Volume Policy allow users to set available performance tunings based on their workload. Below are the tunings that can be configured

- `queueDepth`:
cStor target `queueDepth`, This limits the ongoing IO count from iscsi client on Node to cStor target pod. Default value is 32.

- `luworkers`
cStor target IO worker threads, sets the number of threads that are working on `QueueDepth` queue.
Default value is `6`. In case of better number of cores and RAM, this value can be `16`. 
This means 16 threads will be running for each volume.

- `zvolWorkers`:
cStor volume replica IO worker threads, defaults to the number of cores on the machine.
In case of better number of cores and RAM, this value can be `16`.


```yaml
apiVersion: cstor.openebs.io/v1
kind: CStorVolumePolicy
metadata:
  name: csi-volume-policy
  namespace: openebs
spec:
  replica:
    zvolWorkers: "4"
  target:
    luWorkers: 6
    queueDepth: "32"
```

Note: These Policy tunable configurations can be changed for already provisioned volumes by editing the corresponding volume CStorVolumeConfig resources.

### Resource Request and Limits:

CStorVolumePolicy can be used to configure the volume Target pod resources requests and
limits to ensure QOS. Below is the example to configure the target container resources
requests and limits, as well as auxResources configuration for the sidecar containers.

Learn more about (Resources configuration)[https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/]

```yaml
apiVersion: cstor.openebs.io/v1
kind: CStorVolumePolicy
metadata:
  name: csi-volume-policy
  namespace: openebs
spec:
  target:
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
    auxResources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
```

Note: These resource configuration can be changed once the volume get provisioned by editing the CStorVolumeConfig resource on per volume level.

E.g. You can apply a patch to update CStorVolumeConfig resource that already exists.  Create a file that contains the changes like `patch-resources-cvc.yaml`

```yaml
spec:
  policy:
    target:
      resources:
        limits:
          cpu: 500m
          memory: 128Mi
        requests:
          cpu: 250m
          memory: 64Mi
      auxResources:
        limits:
          cpu: 500m
          memory: 128Mi
        requests:
          cpu: 250m
          memory: 64Mi
```

and apply the patch on the resource

```bash
kubectl patch cvc -n openebs -p "$(cat patch-resources-cvc.yaml)" pvc-0478b13d-b1ef-4cff-813e-8d2d13bcb316 --type merge
```

### Target Pod Toleration:

This Kubernetes feature allows users to mark a node (taint the node) so that no pods can be scheduled to it, unless a pod explicitly tolerates the taint. 
Using this Kubernetes feature we can label the nodes that are reserved (dedicated) for specific pods.

E.g. all the volume specific pods in order to operate flawlessly should be scheduled to nodes that are reserved for storage.

```yaml
apiVersion: cstor.openebs.io/v1
kind: CStorVolumePolicy
metadata:
  name: csi-volume-policy
  namespace: openebs
spec:
  replica: {}
  target:
    tolerations:
    - key: "key1"
      operator: "Equal"
      value: "value1"
      effect: "NoSchedule"
```

### Target Pod NodeSelector:

This feature allows user to specify a set node labels(valid labels) on `policy.spec.target.nodeSelector`, so that target pod will get scheduled only on specified set of nodes(An example case is where user dedicates set of nodes in a cluster for storage `kubectl label node <node-1> <node-2>...<node-n> openebs.io/storage=true`).

#### Influence Target Pod Scheduling During Volume Provisioning Time:

Ex: Create CStorVolumePolicy by populating values under `policy.spec.target.nodeSelector`

```yaml
apiVersion: cstor.openebs.io/v1
kind: CStorVolumePolicy
metadata:
  name: csi-volume-policy
  namespace: openebs
spec:
  replica: {}
  target:
    nodeSelector:
      openebs.io/storage: "true"
```

#### Dynamically Update NodeSelector On CstorVolumeConfig:

cStor also provides an options to dynamically update the nodeSelectors under CStorVolumeConfig.

- Edit/patch the CStorVolumeConfig(CVC) for a particular volume(CVC name will be same as PV name) and add nodeSelectors under `cvc.spec.policy.target.nodeSelector`. Once changes are applied successfully then target pod will get schedule as per specified configuration.

```sh
kubectl edit cvc <cvc_name> -n openebs
```

### Priority Class:

Priority classes can help you control the Kubernetes scheduler decisions to favor higher priority pods over lower priority pods.
The Kubernetes scheduler can even preempt (remove) lower priority pods that are running so that pending higher priority pods can be scheduled.
By setting pod priority, you can help prevent lower priority workloads from impacting critical workloads in your cluster, especially in cases where the cluster starts to reach its resource capacity.

Learn more about (PriorityClasses)[https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#priorityclass]

*NOTE:* Priority class needs to be created before volume provisioning. In this case, `storage-critical` priority classes should exist.

```yaml
apiVersion: cstor.openebs.io/v1
kind: CStorVolumePolicy
metadata:
  name: csi-volume-policy
  namespace: openebs
spec:
  provision:
    replicaAffinity: true
  target:
    priorityClassName: "storage-critical"
```
