## Introduction

Kubernetes support many types of volumes. A Pod can use any number of volume types simultaneously. One of the most useful types of volumes in Kubernetes is `NFS`.  A NFS volume can be accessed from multiple pods at the same time. This is really useful for running applications that need a filesystem that’s shared between multiple application servers.  NFS is a commonly used solution to provide ReadWriteMany(RWX) volumes on block storage in Kubernetes. This server offers a PersistentVolumeClaim (PVC) in RWX mode so that multiple applications can access the data in a shared fashion. In many cases, cloud block storage providers or OpenEBS volumes are used as persistent backend storage for these NFS servers to provide a scalable and manageable RWX shared storage solution. 

OpenEBS Dynamic NFS PV provisioner can be used to dynamically provision NFS Volumes using different kinds of block storage available on the Kubernetes nodes.  OpenEBS NFS provisioner is a kernel based server and thus it requires the NFS related packages has to be preinstalled on the required hosts. 

In this document, we will explain how you can easily set up a NFS solution using OpenEBS block storage in your K8s cluster and provision a scalable WordPress stateful application using this NFS solution.

## Deployment model



<img src="/docs/workload/NFS-Provisioner/RWX-WordPress.svg" alt="OpenEBS and NFS using cStor" style="width:100%;">



We will add a 100G disk to each node. These disks will be consumed by CSI based cStor pool and later we will use this storage as the backend storage for OpenEBS NFS provisioner. The deployment of OpenEBS NFS provisioner will create a storage class where provisioner will be OpenEBS NFS provisioner which uses cStor storage as the backend storage for provisioning persistent volume for NFS based applications. The recommended configuration is to have at least three nodes for provisioning at least 3 volume replicas for each volume and an unclaimed external disk to be attached per node to create cStor storage pool on each node with striped manner .  



## Configuration workflow

1. [Prerequisites](/docs/workload/wordpress/wordpress-rwx.md#prerequisites)

2. [How to use NFS volume for different applications?](/docs/workload/wordpress/wordpress-rwx.md#how-to-use-nfs-volume-for-different-applications?)

    


### Prerequisites

- OpenEBS should be installed and then configure cStor operator. The steps for doing this configuration can be found [here](https://github.com/openebs/cstor-operators/blob/HEAD/docs/quick.md). cStor configuration includes installation of cStor operator, then provisioning of a CStorPoolCluster(CSPC) and finally the creation of a StorageClass that will consume the created CSPC pool. 
- Install OpenEBS NFS provisioner in your cluster. The steps can be found from [here](https://github.com/openebs/cstor-operators/tree/HEAD/docs/tutorial/volumes/rwx-with-nfs.md).
- Install NFS client packages in all worker nodes. In this example, we used base OS as Ubuntu on all worker nodes. The  `nfs-common` packages are installed on all worker nodes and then enabled the NFS service.

### How to use NFS volume for different applications?

As per the the steps mentioned in the prerequisites, we have created a storage class `openebs-rwx` with RWX support.  Any application which uses this NFS supported storage class in it's deployment command, OpenEBS NFS provisioner will create a persistent volume on cStor storage with RWX support.

For example, if a User want to deploy WordPress application in Kubernetes, user can mention this NFS storage class in the WordPress deployment application  command. In this example, `openebs-rwx` is used as the NFS storage class in WordPress application installation.  Run the following command when you have created a namespace `wordpress`.

```
helm install my-release -n wordpress \
       --set wordpressUsername=admin \
       --set wordpressPassword=password \
       --set mariadb.auth.rootPassword=secretpassword \
       --set persistence.storageClass=openebs-rwx \
       --set persistence.accessModes={ReadWriteMany} \
       --set volumePermissions.enabled=true \
       --set autoscaling.enabled=true \
       --set autoscaling.minReplicas=2 \
       --set autoscaling.maxReplicas=6 \
       --set autoscaling.targetCPU=80 \
        bitnami/wordpress
```

The above will create two WordPress application pods with RWX persistent volumes where both WordPress pods can access the same data with shared manner.

Sample output of WordPress application pods:

```
NAME                                    READY   STATUS    RESTARTS   AGE
my-release-mariadb-0                    1/1     Running   0          8m52s
my-release-wordpress-766fcb7546-9q726   1/1     Running   0          8m34s
my-release-wordpress-766fcb7546-pbpgt   1/1     Running   0          8m52s
```
Verify the PVCs created in `wordpress` namespace:
```
kubectl get pvc -n wordpress
```
Sample output:
```
NAME                        STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
data-my-release-mariadb-0   Bound    pvc-3b7b4c44-1026-47d4-b93e-58823a320161   8Gi        RWO            gp2            11s
my-release-wordpress        Bound    pvc-5637dbf4-cafa-4725-a4a8-9a408452cc5a   10Gi       RWX            openebs-rwx    11s
```
From the above output, the WordPress volume `my-release-wordpress` is having `RWX` access mode. So, both pods of WordPress can access the data at the same time.

Verify the PVCs created in `openebs` namespace:
```
kubectl get pvc -n openebs
```
Sample output:
```
NAME                                           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
nfs-pvc-5637dbf4-cafa-4725-a4a8-9a408452cc5a   Bound    pvc-99b40863-ea16-4dc8-9d54-ca5051940625   10Gi       RWO            cstor-csi      32s
```

Verify the PVs created in the cluster:
```
kubectl get pv
```
Sample output:
```
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                                                  STORAGECLASS   REASON   AGE
pvc-3b7b4c44-1026-47d4-b93e-58823a320161   8Gi        RWO            Delete           Bound    wordpress/data-my-release-mariadb-0                    gp2                     9s
pvc-5637dbf4-cafa-4725-a4a8-9a408452cc5a   10Gi       RWX            Delete           Bound    wordpress/my-release-wordpress                         openebs-rwx             15s
pvc-99b40863-ea16-4dc8-9d54-ca5051940625   10Gi       RWO            Delete           Bound    openebs/nfs-pvc-5637dbf4-cafa-4725-a4a8-9a408452cc5a   cstor-csi               15s
```
<br>

## See Also:

### [cStor User guide](https://github.com/openebs/cstor-operators/blob/HEAD/docs/quick.md)

### [Troubleshooting cStor](https://github.com/openebs/cstor-operators/blob/HEAD/docs/troubleshooting/troubleshooting.md)

### [OpenEBS NFS provisioner](https://github.com/openebs/dynamic-nfs-provisioner)

<br>

<hr>
