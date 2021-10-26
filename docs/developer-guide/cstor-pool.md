# cStor Pool Resource Organization

## Introduction

CStor pools can be created on Kubernetes nodes using the CSPC(CStorPoolCluster) custom resource. It is interchangeably also called CSPC API. In order to use a cStor volume and allow it to be consumed by a stateful workload, it is mandatory to create cStor pool cluster first and then cStor volume can be created on top of the created cStor pool(s). 

Before getting into details of CSPC API and how a cStor pool cluster and cStor volume could be created, let us walk through following explanation.


Consider a following kind of setup in a Kubernetes cluster where 3 worker nodes exists and each has 4 disks attached to it.
We could obviously have as many nodes and as many disks attached to a node as practically possible but to understand how cStor pool
stuff works let us keep ourselves to this configuration.

Now the nodes in a Kubernetes cluster is represented by `Node` resource.

Blockdevices(disks) attached to the nodes are represented by a custom resource known as 
`BlockDevice`. This representation is powered by Node-Disk-Manager on which cStor depends for
disk inventory and management.

In a Kubernetes cluster, with OpenEBS installed, you could do `kubectl get blockdevices -n openebs` to
list all the block devices attached to the nodes.


```
+-----------------------+              +-----------------------+               +-----------------------+
|                       |              |                       |               |                       |
|     Worker-Node-1     |              |     Worker-Node-2     |               |     Worker-Node-3     |
|                       |              |                       |               |                       |
|                       |              |                       |               |                       |
++----------------------+              ++----------------------+               ++----------------------+
 |     +---------------+                |     +---------------+                 |     +---------------+
 |     | disk-1(n1)    |                |     | disk-1(n2)    |                 |     | disk-1(n3)    |
 +----->               |                +----->               |                 +----->               |
 |     +---------------+                |     +---------------+                 |     +---------------+
 |                                      |                                       |
 |     +---------------+                |     +---------------+                 |     +---------------+
 |     | disk-2(n1)    |                |     | disk-2(n2)    |                 |     | disk-2(n3)    |
 +----->               |                +----->               |                 +----->               |
 |     +---------------+                |     +---------------+                 |     +---------------+
 |                                      |                                       |
 |     +---------------+                |     +---------------+                 |     +---------------+
 |     | disk-3(n1)    |                |     | disk-3(n2)    |                 |     | disk-3(n3)    |
 +----->               |                +----->               |                 +----->               |
 |     +---------------+                |     +---------------+                 |     +---------------+
 |                                      |                                       |
 |     +---------------+                |     +---------------+                 |     +---------------+
 |     | disk-4(n1)    |                |     | disk-4(n2)    |                 |     | disk-4(n3)    |
 +----->               |                +----->               |                 +----->               |
 |     +---------------+                |     +---------------+                 |     +---------------+
 |                                      |                                       |
 |                                      |                                       |
```

On a high level, the blockdevices on the nodes can be grouped together in a RAID configuration to create a cStor pool.
These cStor pools created on nodes comes together to form a cStor pool cluster.
In cStor, you can only create cStor pool cluster and a cStor pool cluster can have 0 to `k` number of pools. Here
`k` could be any arbitrary practical number.

For example consider following scenarios : 

1. A cStor pool cluster can be created to have a cStor pool on `Worker-Node-1` by using disk `disk-1(n1)` only.
2. A cStor pool cluster can be created to have a cStor pool on `Worker-Node-1` by using disk `disk-1(n1)` and `disk-2(n1)`.
3. A cStor pool cluster can be created to have  cStor pools on `Worker-Node-1`, `Worker-Node-2` and `Worker-Node-3` by taking 2 disks
from each node.

**NOTES:**
Consider following statements: 
- Create cStor pools.
- Create cStor pool cluster.

In common discussion both the above statements could mean the same thing but a cStor user always creates a cStor pool cluster and that 
cStor pool cluster can have minimum 1 cStor pool in order to finally create a volume. Users can decide how many
cStor pools they want to be there in a cStor pool cluster.

To create cStor pool cluster, CSPC custom resource (API) can be used. One need to put the details in a CSPC YAML file, for example nodes and associated 
disks they want to select with other relevant information and then do a `kubectl apply -f <cspc-yaml-file>` to create a cStor pool cluster.
To learn more about how to create pool cluster using CSPC, follow this link. (TODO)


*CSPC is a declarative API where you could specify the intent/configuration of your pool cluster to be configured in a Kubernetes 
cluster and cStor components would converge the system towards the specified intent.* 

## CStor pool Custom Resource and Component

### CSPC-Operator Introduction

This sections describes the custom resources and components involved in cStor pools.

```
                                                              +-------------------------+
                                                              |                         |
                                                              |                         |
                                                              |                         |
                                                              |                         |
                                                              |      CSPC-Operator      |
                                                              |                         |
                               XXXXXXXXXXXXXXX                |                         |
                              X              X                |                         |
Admin/User/SRE               X               X                |                         |
                            X                X  Reconcile     |                         |
     XXX                   X                 +--------------->+                         |
    XXXXX                  X                 X                |                         |
      X                    X   CSPC-YAML     X                |                         |
    XXXXX                  X                 X                |                         |
      X        Applies     X                 X                +-------------------------+
      X     +------------->X                 X
     XXX                   X                 X
    X X X                  X                 X
      X                    X                 X
                           X                 X
                           XXXXXXXXXXXXXXXXXXX

```

1. CSPC contains a list of pool specs and a pool spec contains configuration for cStor pool creation on a particular node as specified in the pool spec.
For example, if a CSPC manifest has 3 pool specs, 3 cStor pools will be created on 3 different nodes. For a CSPC, no two cStor pools will be created on the same node.

2. An admin/SRE can preapre their CSPC YAML according to the requirements and do a `kubectl apply -f <cspc-yaml-file>` to provision a CStorPoolCluster.

3. Once a CSPC is applied, it is validated by cStor admission server and if the validation passes it is admitted to the etcd.

4. The `CSPC-operator` receives events from Kubernetes APIs  for the created CSPC as well as existing CSPC and `CSPC-Operator` converges the system towards intent specified in the CSPC. Just a note that `CSPC-Operator` is deployed as a deployment controller and runs as a pod in the Kubernetes cluster.

### What does CSPC-Operator do?

1. CSPC Operator creates the following resources as part of its job to converge the system towards the intent specified in the CSPC:
- CStorPoolInstance(CSPI) - CSPI is a Kubernetes custom resource like the CSPC.
- Pool Manager Deployment - Deployment is a native Kubernetes resource.
- BlockDeviceClaim(BDC) - BDC is again a custom resource. 

2. Let us say, CSPC has `k` pool specs then exactly `k` CSPI and `k` pool manager deployment is created. Each CSPI created corresponds to only one pool manger. So we can say that CSPI and pool manager has a one to one mapping.

2. Each pool spec specified in the CSPC is converted into a CSPI resource and created to persist in etcd. For each CSPI a corresponding pool manager is also created. 

3. For each block device mentioned in the CSPC YAML for all the pool specs, CSPC-Operator creates a BDC to ensure exclusive ownership over the disks. As discussed earlier, disks are represented as BlockDevice(BD) custom resource. Hence, there is a one to one mapping between BDC and BD.  

3. The way CSPC is reconciled by CSPC-operator, in the same way a pool-manger reconciles its corresponding CSPI resource. It is important to note that a pool manager reconciles only one CSPI that corresponds to the pool manager.

4. Pool-manager is again a Kubernetes deployment that reads the configuration from CSPI and execute ZFS related commands on node to satisfy the CSPI
configuration e.g. creation of pool, deletion of pool, enabling/disabling compression on pool.

Following is a representation of the resources involved for cStor pool. CSPC,CSPI,BD and BDC custom resources facilitates cStor pool provisioning and 
other operations.

While using cStor for pool related operations CSPC API is the only contact point for a cStor user and it is not recommended to modify the other
custom resource.

For more details on how to use CSPC please follow this link.


```
                                                                     +---------------+
                                                       Reconcile     |               |    APIs
                                                    +--------------->+ CSPC-Operator |
                                                    |                |               |
                                                    |                +---------------+
                                                    |
                                                    |
                                           +--------+---------+
             +-----------------------------+      CSPC        +----------------------------+
             |                             +--------+---------+                            |
             |   Reconcile +--------+               |   Reconcile +--------+               |   Reconcile +--------+
             |   +-------->+        |               |   +--------->        |               |   +-------->+        |
             v   |         |  Pool  |               v   |         |  Pool  |               v   |         |  Pool  |
    +--------+---+-----+   | Manager|      +--------+---+-----+   | Manager|      +--------+---+-----+   | Manager|
    |      CSPI        |   |        |      |      CSPI        |   |        |      |      CSPI        |   |        |
    +--------+---------+   +--------+      +--------+---------+   +--------+      +--------+---------+   +--------+
             |                                      |                                      |
+------------v--------------+          +------------v--------------+          +------------v--------------+
|                           |          |                           |          |                           |
|                           |          |                           |          |                           |
|          Node             |          |          Node             |          |          Node             |
|                           |          |                           |          |                           |
|                           |          |                           |          |                           |
+---^---------^---------^---+          +---^---------^---------^---+          +---^---------^---------^---+
    |         |         |                  |         |         |                  |         |         |
 +--+--+   +--+--+   +--+--+            +--+--+   +--+--+   +--+--+            +--+--+   +--+--+   +--+--+
 | BD  |   | BD  |   | BD  |            | BD  |   | BD  |   | BD  |            | BD  |   | BD  |   | BD  |
 +--+--+   +--+--+   +--+--+            +--+--+   +--+--+   +--+--+            +--+--+   +--+--+   +--+--+
    |         |         |                  |         |         |                  |         |         |
 +--+--+   +--+--+   +--+--+            +--+--+   +--+--+   +--+--+            +--+--+   +--+--+   +--+--+
 | BDC |   | BDC |   | BDC |            | BDC |   | BDC |   | BDC |            | BDC |   | BDC |   | BDC |
 +-----+   +-----+   +-----+            +-----+   +-----+   +-----+            +-----+   +-----+   +-----+

```

To understand cStor volume resource organization, please follow this [link](cstor-volume.md).

*CSPC is an declarative API that enables you to create a group of cStor pool(s) on Kubernetes nodes as well as allow you to do other pool operations e.g. disk replacement, pool expansion etc in a Kubectl native way. CSPC API also supports GitOps model.*
