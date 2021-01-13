### Topology Aware StorageClasses

There can be a use case where we have a certain set of nodes that we want to
dedicate as storage Nodes and wants to deploy the CSI Node driver on that nodes
and later we want a particular type of application to use those Nodes.

We can create a storage class with `allowedTopologies` and mention all the storage
nodes :


```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: cstor-sparse-auto
provisioner: cstor.csi.openebs.io
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
parameters:
  replicaCount: "3"
  cstorPoolCluster: "cspc-ssd"
  cas-type: "cstor"
allowedTopologies:
- matchLabelExpressions:
  - key: "topology.cstor.openebs.io/nodeName"
    values:
      - node-1
      - node-2
      - node-3
```

The CStor CSI driver will create the Volume in the CStor Pool Instances present on
this list of nodes. We can use `volumeBindingMode: WaitForFirstConsumer` to let the k8s select
the node where the volume should be provisioned.

With `volumeBindingMode: Immediate` driver creates the volume without any topology awareness.
Volume binding and dynamic provisioning are handled when the PVC is created. This is the default
VolumeBindingMode and is suited for clusters that do not enforce topology constraints.
in such cases `nodeAffinity` can be used in the application pod to select the desire nodes.

The problem with the above StorageClass is that it works fine if the number of nodes
is less, but if the number of nodes is huge, it is cumbersome to list all the nodes
like this. In that case, we can label all the similar nodes using the same key value
and use that label as `allowedTopologies` to create the StorageClass.

```
$ kubectl label node node-1 node-2 node-3 company/nodegroup=storage
```

Now, restart the CStor csi-node Driver pods (if already deployed, otherwise please ignore)
so that it can update the new node label as the supported topology.

```sh
$ kubectl delete pods -n openebs -l role=openebs-cstor-csi
```

Now, we can create the StorageClass like this:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: cstor-sparse-auto
provisioner: cstor.csi.openebs.io
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
parameters:
  replicaCount: "3"
  cstorPoolCluster: "cspc-ssd"
  cas-type: "cstor"
allowedTopologies:
- matchLabelExpressions:
  - key: "company/nodegroup"
    values:
      - storage
```
