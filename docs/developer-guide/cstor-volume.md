# cStor Volume Resource Organization

## Introduction

It is recommended to go through the cStor Pool resource organisation [doc](cstor-pool.md) before reading this document.

CStor CSI volumes are created by creating a PVC in a Kubernetes cluster which references a CSPC based OpenEBS storage 
class. A CSPC based storage class is simply a storage class that refers a CStorPoolCluster. A CStor CSI volume is 
collection of 1 or more volume replica that gets distributed across the CStorPoolInstances of a CStorPoolCluster. In this 
sense, CStor volumes provide a replicated storage.

To understand more on how CSI works please refer to the following document: 

https://github.com/container-storage-interface/spec/blob/master/spec.md

Following are the list of custom resources that enables the cStor CSI volume features:
- CStorVolume(cv)
- CStorVolumeReplica(cvr)
- CStorVolumeConfig(cvc)
- CStorVolumePolicy(cvp,policy)
- CStorVolumeAttachment(cva)

Following are the list of native Kubernetes resources that enables the cStor CSI volume features:
- Service (Known as target service)
- Deployment (Known as cStor Target)

# cStor CSI Driver Overview

OpenEBS cStor CSI driver(or plugin) follows a cetralized plugin architecture. In centralized plugin architeture, two 
components are deployed in following manner:
1. **Controller Plugin:** It is deployed as a sidecar container in statefulset with other CSI specific containers.

2. **Node Plugin:** It is deployed as a daemonset controller. 

Controller plugin and node plugin both run a set of containers inside their pod. Controller plugin carries out volume
orchestration related tasks and node plugin actually helps in executing the relevant volume operations as required on
the node.

The `Controller Plugin` pod has following containers: 
- csi-resizer
- csi-snapshotter
- snapshot-controller
- csi-provisioner
- csi-attacher
- csi-cluster-driver-registrar
- cstor-csi-plugin

The `Node Plugin` pod has following containers: 
- csi-node-driver-registrar
- cstor-csi-plugin

The `cstor-csi-plugin` container (as you can see) is present in both the `Controller Plugin` and `Node Plugin` pods. 
In `Controller Plugin` the `cstor-csi-plugin` acts as a controller driver and in `Node Plugin` it acts as a node
driver which is determined via a flag that is passed.

This document focuses on understanding the custom and native resources organisation for a cStor CSI volume which is
discussed in the next section.

## Resource Organisation

This section gives an overview of how a cStor CSI volume looks from the OpenEBS side:

1. When a PVC is created and persisted in etcd, the csi-provisioner container of the controller plugin pod reconciles 
it to provision a cStor CSI volume. It should be noted that a PVC references a cStor CSI storage class and the storage
class in turns references a CStorPoolCluster where the placement of volume replica will happen. 

```
                                                              +-------------------------+
                                                              |   Controller Plugin     |
                                                              +-------------------------+
                                                              |                         |
                                                              |                         |
                                                              |         csi-            |
                                                              |      provisioner        |
                               XXXXXXXXXXXXXXX                |                         |
                              X              X                |                         |
Admin/User/SRE               X               X                |                         |
                            X                X  Reconciles    |                         |
     XXX                   X                 +--------------->+                         |
    XXXXX                  X                 X                |                         |
      X                    X   PVC-YAML      X                |                         |
    XXXXX                  X                 X                |                         |
      X        Applies     X                 X                +-------------------------+
      X     +------------->X                 X
     XXX                   X                 X
    X X X                  X                 X
      X                    X                 X
                           X                 X
                           XXXXXXXXXXXXXXXXXXX

```
2. As part of PVC create, the csi-provisioner(of Controller Plugin pod) executes a gRPC method that is implemented by 
`cstor-csi-plugin` (of controller plugin pod). This method is `CreateVolume`.

3. The `cstor-csi-plugin` (as part of CreateVolume) creates a CStorVolumeConfig(CVC) custom resource that is reconciled 
by CVC-Operator. It should be noted that CVC-Operator is deployed as part of cstor-operators installation.

```
                                                                    +------------------+
                                                                    |                  |
+--------------------+                                              |                  |
| Controller Plugin  |                                              |                  |
|                    |                                              |                  |
+--------------------+  cstor-csi-plugin  XXXXXXXXX                 |                  |
|                    |  creates CVC as   X        X                 |   CVC-Operator   |
|                    |  part of volume  X         X                 |                  |
| cstor-csi-plugin   |  create request X          X   Reconciles    |                  |
|                    |                 X          +---------------->+                  |
|                    +---------------->X   CVC    X                 |                  |
|                    |                 X          X                 |                  |
|                    |                 X          X                 |                  |
|                    |                 X          X                 +------------------+
|                    |                 XXXXXXXXXXXX
+--------------------+

```

4. Now the CVC-Operator creates following resources as part of CVC reconciliation:
- A `CStorVolume` (CV) (custom resource).  
- `k` number of `CStorVolumeReplica` (CVR) (custom resource) where `k` is the replica count specified in the 
StorageClass parameters.
- A deployment known as `cStor target`.
- A service. 

5. `CStorVolume` resource is reconciled by the corresponding `cStor target` that got created.

6. The CVR is labelled with a CSPI UUID , that means replicas will be distributed across the pools. No two 
CStorVolumeReplicas for the given PVC will be placed on the same CStorPoolInstance(CSPI).

7. Now, pool manager pod have a replica controller running which reconciles for the CVR and executes the `ZFS` 
commands after reading the configuration from the CVR.

```





                  +------------------+                        +------------------+          +------------------+
                  |                  |     Reconciles         |                  |          |                  |
                  |                  +<-----------------------+       PVC        +--------->+       SC         |
                  |   Controller     |                        |                  |          |                  |
                  |    Plugin        |                        +------------------+          +------------------+
                  |                  |                                 |
                  |                  |                                 |
                  |                  |                                 |                                 +--------------------+
                  +------------------+                                 |                                 |                    |
                                                              +--------v---------+     Reconciles        |                    |
                                                              |                  |                       |   CVC-Operator     |
                                                              |       CVC        +---------------------->+                    |
                                                              |                  |                       |                    |
                                                              +------------------+                       |                    |
                                                                       |                                 +--------------------+
                                                                       |
                                                                       |
                                                                       |
                                                                       |
                                                                       v
                                                              +------------------+
                                                              |                  |
                    +-----------------------------------------+       CV         +-------------------------------------------+
                    |                                         |                  |                                           |
                    |                                         +------------------+                                           |
                    |                                                  |                                                     |
                    |                                                  |                                                     |
                    |                                                  v                                                     |
           +--------v---------+     +--------------+          +------------------+     +--------------+             +--------v---------+     +--------------+
           |                  |     |    Pool      |          |                  |     |    Pool      |             |                  |     |    Pool      |
           |       CVR        +-+   |  Manager     |          |       CVR        +-+   |  Manager     |             |       CVR        +-+   |  Manager     |
           |                  | |   +--------------+          |                  | |   +--------------+             |                  | |   +--------------+
           +------------------+ |   |   Replica    |          +------------------+ |   |   Replica    |             +------------------+ |   |   Replica    |
                    |           +-->+  Controller  |                   |           +-->+  Controller  |                      |           +--->  Controller  |
                    |               |              |                   |               |              |                      |               |              |
                    v               +------------->+                   v               +-------------->                      v               +------------->+
           +------------------+     |    Pool      |          +------------------+     |    Pool      |             +------------------+     |    Pool      |
           |                  |     |  Controller  |          |                  |     |  Controller  |             |                  |     |  Controller  |
           |       CSPI       +---->+              |          |       CSPI       +---->+              |             |       CSPI       +---->+              |
           |                  |     |              |          |                  |     |              |             |                  |     |              |
           +------------------+     +--------------+          +------------------+     +--------------+             +------------------+     +--------------+

                  CVR is reconciled by replica controller routine of pool manager pod and CSPI by the pool controller 
                  routine. Note that replica controller and pool controller are two different routines in the same 
                  container known as cstor-pool-mgmt.

``` 

8. `Node plugin` component is not shown in the above diagram to keep it simple. But the work of node plugin is to
finally make the volume available for a pod scheduled on the node by staging and publishing it.
