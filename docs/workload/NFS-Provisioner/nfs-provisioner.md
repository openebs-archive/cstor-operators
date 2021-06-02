## Introduction

Kubernetes support many types of volumes. A Pod can use any number of volume types simultaneously. One of the most useful types of volumes in Kubernetes is `NFS`.  A NFS volume can be accessed from multiple pods at the same time. This is really useful for running applications that need a filesystem that’s shared between multiple application servers.  NFS is a commonly used solution to provide ReadWriteMany(RWX) volumes on block storage in Kubernetes. This server offers a PersistentVolumeClaim (PVC) in RWX mode so that multiple applications can access the data in a shared fashion. In many cases, cloud block storage providers or OpenEBS volumes are used as persistent backend storage for these NFS servers to provide a scalable and manageable RWX shared storage solution. 

OpenEBS Dynamic NFS PV provisioner can be used to dynamically provision NFS Volumes using different kinds of block storage available on the Kubernetes nodes.  OpenEBS NFS provisioner is a kernel based server and thus it requires the NFS related packages has to be preinstalled on the required hosts. 

In this document, we will explain how you can easily set up a NFS solution using OpenEBS block storage in your K8s cluster.

## Deployment model



<img src="/docs/workload/NFS-Provisioner/RWX-WordPress.svg" alt="OpenEBS and NFS using cStor" style="width:100%;">



We will add a 100G disk to each node. These disks will be consumed by CSI based cStor pool and later we will use this storage as the backend storage for OpenEBS NFS provisioner. The deployment of OpenEBS NFS provisioner will create a storage class where provisioner will be OpenEBS NFS provisioner which uses cStor storage as the backend storage for provisioning persistent volume for NFS based applications. The recommended configuration is to have at least three nodes for provisioning at least 3 volume replicas for each volume and an unclaimed external disk to be attached per node to create cStor storage pool on each node with striped manner .  



## Configuration workflow

1. [Meet Prerequisites](/docs/workload/NFS-Provisioner/nfs-provisioner.md#meet-prerequisites)

4. [Installing OpenEBS NFS Provisioner](/docs/workload/NFS-Provisioner/nfs-provisioner.md#installing-openebs-nfs-provisioner)

5. [How to use NFS volume for different applications?](/docs/workload/NFS-Provisioner/nfs-provisioner.md#how-to-use-nfs-volume-for-different-applications?)

   

### Meet Prerequisites

- OpenEBS should be installed first on your Kubernetes cluster. The steps for OpenEBS installation can be found [here](https://docs.openebs.io/docs/next/installation.html). 

- After OpenEBS installation, choose the OpenEBS storage engine as per your requirement. 
  - Choose **cStor**, If you are looking for replicated storage feature and other enterprise graded features such as volume expansion, backup and restore, etc. cStor configuration can be found [here](https://github.com/openebs/cstor-operators/blob/master/docs/quick.md). In this document, we are mentioning about the installation of OpenEBS NFS provisioner using cStor operator.
  - Choose **OpenEBS Local PV**, if you are not requiring replicated storage but high performance storage engine.
- Install NFS client packages in all worker nodes. In this example, we used the base OS as Ubuntu fin worker nodes. The  `nfs-common` packages are installed on all worker nodes and then enabled the NFS service.

### Installing OpenEBS NFS Provisioner

In this section, we will install the OpenEBS NFS provisioner where OpenEBS cStor storage engine is used as the backend storage. The following command will fetch the OpenEBS NFS provisioner YAML spec and user can provide the storage class of required block storage as the `BackendStorageClass`. 

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
NAME                                       READY   STATUS    RESTARTS   AGE
openebs-nfs-provisioner-7b4c9b87d9-s7xwv   1/1     Running   0          32s
```

Verify if NFS supported new StorageClass is created successfully:

```
kubectl get sc
```

Sample output:

```
NAME                        PROVISIONER                                                RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
cstor-csi                   cstor.csi.openebs.io                                       Delete          Immediate              true                   3m14s
gp2 (default)               kubernetes.io/aws-ebs                                      Delete          WaitForFirstConsumer   false                  34m
openebs-device              openebs.io/local                                           Delete          WaitForFirstConsumer   false                  18m
openebs-hostpath            openebs.io/local                                           Delete          WaitForFirstConsumer   false                  18m
openebs-jiva-default        openebs.io/provisioner-iscsi                               Delete          Immediate              false                  18m
openebs-rwx                 openebs.io/nfsrwx                                          Delete          Immediate              false                  2m46s
openebs-snapshot-promoter   volumesnapshot.external-storage.k8s.io/snapshot-promoter   Delete          Immediate              false                  18m
```

From the above output, `openebs-rwx` is the storage class that supports shared storage using OpenEBS NFS provisioner. So in this cluster, any application which uses `openebs-rwx` storage class, it will create a persistent volume on cStor storage with NFS support.

**Note:** Don’t forget to install NFS client packages on all worker nodes.  If NFS client packages are not installed and enabled then it will fail to provision any application which uses the NFS storage class.

### How to use NFS volume for different applications?

Any application which uses above created NFS storage class(In this example `openebs-rwx` storage class) in it's deployment command, OpenEBS NFS provisioner will create a persistent volume on cStor storage with RWX support.

For example, if a user want to deploy WordPress application in Kubernetes, user can mention this NFS storage class in the WordPress deployment application  command. In this example, `openebs-rwx` is used as the NFS storage class in WordPress application installation. 

```
helm install my-release -n wordpress \
       --set wordpressUsername=admin \
       --set wordpressPassword=password \
       --set mariadb.auth.rootPassword=secretpassword \
       --set global.storageClass=openebs-rwx \
       --set autoscaling.enabled=true \
       --set autoscaling.minReplicas=2 \
       --set autoscaling.maxReplicas=6 \
       --set autoscaling.targetCPU=80 \
        bitnami/wordpress
```

The above will create two WordPress application pods with RWX persistent volumes where both WordPress pods can access the same data with shared manner.

Sample output of WordPress application pods:

```
NAME                                                        	      READY       STATUS           RESTARTS       AGE
my-release-mariadb-0                                                  1/1 	  Running          0      	  7m42s
my-release-wordpress-5fb5d559f8-t7fwr                                 1/1 	  Running          0      	  7m27s 
my-release-wordpress-5fb5d559f8-v7phz                                 1/1 	  Running          0      	  7m42s 
nfs-pvc-1920eb0c-dd81-4d5c-9ee0-e139f052bcde-cccdf9676-wp7xg	      1/1 	  Running          0      	  7m43s
nfs-pvc-b70fc474-c3fe-401d-9f49-330be63f9f0d-84bcf4c79b-md4qg         1/1 	  Running          0      	  7m42s
openebs-nfs-provisioner-787fc9c5d8-8cmwt                              1/1 	  Running          0      	  80m
```
<br>

## See Also:

### [cStor User guide](https://github.com/openebs/cstor-operators/blob/master/docs/quick.md)

### [Troubleshooting cStor](https://github.com/openebs/cstor-operators/blob/master/docs/troubleshooting/troubleshooting.md)

### [OpenEBS NFS provisioner](https://github.com/openebs/dynamic-nfs-provisioner)

<br>

<hr>
