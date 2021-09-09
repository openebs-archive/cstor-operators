## Introduction

Kubernetes support many types of volumes. A Pod can use any number of volume types simultaneously. One of the most useful types of volumes in Kubernetes is `NFS`.  A NFS volume can be accessed from multiple pods at the same time. This is useful for running applications that need a filesystem that’s shared between multiple application servers.  NFS is a commonly used solution to provide ReadWriteMany(RWX) volumes on block storage in Kubernetes. This server offers a PersistentVolumeClaim (PVC) in RWX mode so that multiple applications can access the data in a shared fashion. In many cases, cloud block storage providers or OpenEBS volumes are used as persistent backend storage for these NFS servers to provide a scalable and manageable RWX shared storage solution. 

OpenEBS Dynamic NFS PV provisioner can be used to dynamically provision NFS Volumes using different kinds of block storage available on the Kubernetes nodes. OpenEBS NFS provisioner is a kernel based server and thus it requires the NFS related packages to be preinstalled on the required hosts. 

In this document, we will explain how you can easily set up a NFS solution using OpenEBS block storage in your K8s cluster.  We will add a 100G disk to each node. These disks will be consumed by CSI based cStor pool and later we will create a cStor storage class and use this storage class as the backend storage for OpenEBS NFS provisioner. The deployment of OpenEBS NFS provisioner will create a storage class where the provisioner will be OpenEBS NFS provisioner which uses cStor storage as the backend storage for provisioning persistent volume for NFS based applications. The recommended configuration is to have at least three nodes for provisioning at least 3 volume replicas for each volume and an unclaimed external disk to be attached per node to create a cStor storage pool on each node in a striped manner.



## Configuration workflow

1. [Prerequisites](/docs/tutorial/volumes/rwx-with-nfs.md#prerequisites)

2. [Installing OpenEBS NFS Provisioner](/docs/tutorial/volumes/rwx-with-nfs.md#installing-openebs-nfs-provisioner)


### Prerequisites

- OpenEBS should be installed and then configure cStor operator. The steps for doing this configuration can be found [here](https://github.com/openebs/cstor-operators/blob/HEAD/docs/quick.md). cStor configuration includes installation of cStor operator, then provisioning of a CStorPoolCluster(CSPC) and finally the creation of a StorageClass that will consume the created CSPC pool. 
- Install NFS client packages on all worker nodes before deploy any application with the NFS supported StorageClass. In this example, we used base OS as `Ubuntu` on all worker nodes. The  `nfs-common` packages are installed on all worker nodes and then enabled the NFS service.

### Installing OpenEBS NFS Provisioner

In this section, we will install the OpenEBS NFS provisioner where the OpenEBS cStor storage engine is used as the backend storage. As part of the pre-requisite, we have created a cStor storage class named `cstor-csi`. The following command will fetch the OpenEBS NFS provisioner YAML spec and the user can provide the required storage class as the `BackendStorageClass`. 

Get the OpenEBS NFS provisioner manifest:

```
wget https://raw.githubusercontent.com/openebs/dynamic-nfs-provisioner/develop/deploy/kubectl/openebs-nfs-provisioner.yaml
```

Modify the storage class section by uncommenting the `BackendStorageClass` and its `value` and add the corresponding storage class name. In this example, we are using `cstor-csi` as the backend storage for OpenEBS NFS provisioner.

Sample storage class for NFS provisioner:

```
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: openebs-rwx
  annotations:
    openebs.io/cas-type: nfsrwx
    cas.openebs.io/config: |
      - name: NFSServerType
        value: "kernel"
      - name: BackendStorageClass
        value: "cstor-csi"
provisioner: openebs.io/nfsrwx
reclaimPolicy: Delete
```

Apply the modified [openebs-nfs-provisioner.yaml](https://raw.githubusercontent.com/openebs/dynamic-nfs-provisioner/develop/deploy/kubectl/openebs-nfs-provisioner.yaml) specification.

```
kubectl apply -f openebs-nfs-provisioner.yaml
```

Verify OpenEBS NFS provisioner is running:

```
kubectl get pod -n openebs -l name=openebs-nfs-provisioner
```

Sample output:

```
NAME                                       READY   STATUS    RESTARTS   AGE
openebs-nfs-provisioner-7b4c9b87d9-fvb4z   1/1     Running   0          69s
```

Verify if NFS supported new StorageClass is created successfully:

```
kubectl get sc
```

Sample output:

```
NAME                        PROVISIONER                                                RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
cstor-csi                   cstor.csi.openebs.io                                       Delete          Immediate              true                   7m6s
gp2 (default)               kubernetes.io/aws-ebs                                      Delete          WaitForFirstConsumer   false                  78m
openebs-device              openebs.io/local                                           Delete          WaitForFirstConsumer   false                  68m
openebs-hostpath            openebs.io/local                                           Delete          WaitForFirstConsumer   false                  68m
openebs-jiva-default        openebs.io/provisioner-iscsi                               Delete          Immediate              false                  68m
openebs-rwx                 openebs.io/nfsrwx                                          Delete          Immediate              false                  85s
openebs-snapshot-promoter   volumesnapshot.external-storage.k8s.io/snapshot-promoter   Delete          Immediate              false                  68m
```

From the above output, `openebs-rwx` is the storage class that supports shared storage using OpenEBS NFS provisioner. So in this cluster, any application which uses `openebs-rwx` storage class, it will create a persistent volume on cStor storage with NFS support.

**Note:** Don’t forget to install NFS client packages on all worker nodes.  If NFS client packages are not installed & enabled, then it will fail to provision any application which uses the above NFS storage class.

<hr>

