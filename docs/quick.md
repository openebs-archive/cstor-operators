# Quickstart

## Prerequisites

Before setting up cStor operators make sure your Kubernetes cluster
meets the following prerequisites:

1. Kubernetes version 1.17 or higher
2. iSCSI initiator utils installed on all the worker nodes (If you are using a Rancher-based cluster, perform the steps mentioned [here](troubleshooting/rancher_prerequisite.md)).


| OPERATING SYSTEM | iSCSI PACKAGE         | Commands to install iSCSI                                | Verify iSCSI Status         |
| ---------------- | --------------------- | -------------------------------------------------------- | --------------------------- |
| RHEL/CentOS      | iscsi-initiator-utils | <ul><li>sudo yum install iscsi-initiator-utils -y</li><li>sudo systemctl enable --now iscsid</li></ul> | sudo systemctl status iscsid.service |
| Ubuntu/Debian   | open-iscsi            |  <ul><li>sudo apt install open-iscsi -y</li><li>sudo systemctl enable --now iscsid</li></ui>| sudo systemctl status iscsid.service |
| RancherOS        | open-iscsi            |  <ul><li>sudo ros s enable open-iscsi</li><li>sudo ros s up open-iscsi</li></ui>| ros service list iscsi |

3. You have disks attached to nodes to provision the storage. The disks MUST not have any filesystem and the disks MUST not be mounted on the Node. cStor requires raw block devices. You can use the `lsblk -fa` command to check if the disks have a filesystem or if the disk is mounted.

<h2 style="color:red;"> CAUTION: </h2>

Follow below practice while running cStor along with kernel ZFS on the same set of nodes
- Disable zfs-import-scan.service service that will avoid importing all pools by scanning all the available devices in the system during boot time, disabling scan service will avoid importing pools that are not created by kernel. Disabling scan service will not cause harm since zfs-import-cache.service is enabled and it is the best way to import pools by looking at cache file during boot time.
  ```sh
  sudo systemctl stop zfs-import-scan.service
  sudo systemctl disable zfs-import-scan.service
  ```
- Always maintain upto date /etc/zfs/zpool.cache while performing operations any day2 operations on zfs pools(zpool set cachefile=/etc/zfs/zpool.cache <pool dataset name>).

Note: Following above two step kernel ZFS will not import the pools created by cStor



## Install


Check for existing NDM components in your openebs namespace. Execute the following command:
```bash
$ kubectl -n openebs get pods -l openebs.io/component-name=ndm

NAME                                                              READY   STATUS    RESTARTS   AGE
openebs-ndm-gctb7                                                 1/1     Running   0          6d7h
openebs-ndm-sfczv                                                 1/1     Running   0          6d7h
openebs-ndm-vgdnv                                                 1/1     Running   0          6d6h
```

If you have got an output as displayed above, then it is recommended that you proceed with installation using the [CStor operators helm chart](https://openebs.github.io/cstor-operators). You will have to exclude `openebs-ndm` charts from the installation. Sample command:
```bash
helm install openebs-cstor openebs-cstor/cstor -n openebs --set openebsNDM.enabled=false
```
<details>
  <summary>Click here if you're using MicroK8s.</summary>

  ```bash
  microk8s helm3 install openebs-cstor openebs-cstor/cstor -n openebs --set-string csiNode.kubeletDir="/var/snap/microk8s/common/var/lib/kubelet/" --set openebsNDM.enabled=false
  ```
</details>

If you did not get any meaningful output (as above), then you do not have NDM components installed. Proceed with any one of the installation options below.

### Using Helm Charts:
 
Install CStor operators and CSI driver components using the [CStor Operators helm charts](https://openebs.github.io/cstor-operators). Sample command:

```bash
helm install openebs-cstor openebs-cstor/cstor -n openebs --create-namespace
```
<details>
  <summary>Click here if you're using MicroK8s.</summary>

  ```bash
  microk8s helm3 install openebs-cstor openebs-cstor/cstor -n openebs --create-namespace --set-string csiNode.kubeletDir="/var/snap/microk8s/common/var/lib/kubelet/"
  ```
</details>


[Click here](https://github.com/openebs/cstor-operators/blob/HEAD/deploy/helm/charts/README.md) for detailed instructions.

### Using Operator:

Install the latest release using CStor Operator yaml.

```bash
kubectl apply -f https://openebs.github.io/charts/cstor-operator.yaml
```
<details>
  <summary>Click here if you're using MicroK8s.</summary>

  ```bash
  microk8s kubectl apply -f https://openebs.github.io/charts/microk8s-cstor-operator.yaml
  ```
</details>


### Local Development:

Alternatively, you may also install the development version  of CStor Operators using:

```bash
$ git clone https://github.com/openebs/cstor-operators.git
$ cd cstor-operators
$ kubectl create -f deploy/yamls/rbac.yaml
$ kubectl create -f deploy/yamls/ndm-operator.yaml
$ kubectl create -f deploy/crds
$ kubectl create -f deploy/yamls/cspc-operator.yaml
$ kubectl create -f deploy/yamls/csi-operator.yaml
```

 **Note: If running on K8s version lesser than 1.17, you will need to comment the `priorityClassName: system-cluster-critical` in the csi-operator.yaml**
 
Once installed using any of the above methods, verify that all NDM and CStor operators pods are running. 

```bash
$ kubectl get pod -n openebs

NAME                                                              READY   STATUS    RESTARTS   AGE
cspc-operator-5fb7db848f-wgnq8                                    1/1     Running   0          6d7h
cvc-operator-7f7d8dc4c5-sn7gv                                     1/1     Running   0          6d7h
openebs-cstor-admission-server-7585b9659b-rbkmn                   1/1     Running   0          6d7h
openebs-cstor-csi-controller-0                                    7/7     Running   0          6d7h
openebs-cstor-csi-node-dl58c                                      2/2     Running   0          6d7h
openebs-cstor-csi-node-jmpzv                                      2/2     Running   0          6d7h
openebs-cstor-csi-node-tfv45                                      2/2     Running   0          6d7h
openebs-ndm-gctb7                                                 1/1     Running   0          6d7h
openebs-ndm-operator-7c8759dbb5-58zpl                             1/1     Running   0          6d7h
openebs-ndm-sfczv                                                 1/1     Running   0          6d7h
openebs-ndm-vgdnv                                                 1/1     Running   0          6d6h
```

Check that blockdevices are created:

```bash
$ kubectl get bd -n openebs

NAME                                           NODENAME           SIZE          CLAIMSTATE   STATUS   AGE
blockdevice-01afcdbe3a9c9e3b281c7133b2af1b68   worker3            21474836480   Unclaimed    Active   2m10s
blockdevice-10ad9f484c299597ed1e126d7b857967   worker1            21474836480   Unclaimed    Active   2m17s
blockdevice-3ec130dc1aa932eb4c5af1db4d73ea1b   worker2            21474836480   Unclaimed    Active   2m12s
```

NOTE:
1. It can take little while for blockdevices to appear when the application is warming up.
2. For a blockdevice to appear, you must have disks attached to node.


## Provision a CStorPoolCluster

For simplicity, this guide will provision a stripe pool on three nodes. A minimum of 3 replicas (on 3 nodes) is recommended for high-availability.

1. Use the CSPC file from [examples/cspc/cspc-single.yaml](/examples/cspc/cspc-single.yaml) and modify by performing
follwing steps:

   Modify CSPC to add your node selector for the node where you want to provision the pool.
   
   List the nodes with labels:

   ```bash
   kubectl get node --show-labels
   ```
   
   ```bash
   NAME               STATUS   ROLES    AGE    VERSION   LABELS
   master1            Ready    master   5d2h   v1.18.0   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/arch=amd64,kubernetes.io/hostname=master1,kubernetes.io/os=linux,node-role.kubernetes.io/master=

   worker1            Ready    <none>   5d2h   v1.18.0   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/arch=amd64,kubernetes.io/hostname=worker1,kubernetes.io/os=linux

   worker2            Ready    <none>   5d2h   v1.18.0   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/arch=amd64,kubernetes.io/hostname=worker2,kubernetes.io/os=linux

   worker3            Ready    <none>   5d2h   v1.18.0   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/arch=amd64,kubernetes.io/hostname=worker3,kubernetes.io/os=linux

   ```
   
   In this guide, worker1 is picked. Modify the CSPC yaml to use this worker.
   (Note: Use the value from labels kubernetes.io/hostname=worker1 as this label value and node name could be different in some platforms)

   ```yaml
   kubernetes.io/hostname: "worker1"
   ```

   Modify CSPC to add blockdevice attached to the same node where you want to provision the pool.
   
   ```bash
   kubectl get bd -n openebs
   ```
   
   ```bash
   NAME                                           NODENAME           SIZE          CLAIMSTATE   STATUS   AGE
   blockdevice-01afcdbe3a9c9e3b281c7133b2af1b68   worker3            21474836480   Unclaimed    Active   2m10s
   blockdevice-10ad9f484c299597ed1e126d7b857967   worker1            21474836480   Unclaimed    Active   2m17s
   blockdevice-3ec130dc1aa932eb4c5af1db4d73ea1b   worker2            21474836480   Unclaimed    Active   2m12s
   ```
    
   ```yaml
   - blockDeviceName: "blockdevice-10ad9f484c299597ed1e126d7b857967"
   ```
   
   Finally the CSPC YAML looks like the following :
   ```yaml
   apiVersion: cstor.openebs.io/v1
   kind: CStorPoolCluster
   metadata:
     name: cstor-storage
     namespace: openebs
   spec:
     pools:
       - nodeSelector:
           kubernetes.io/hostname: "worker-1"
         dataRaidGroups:
           - blockDevices:
               - blockDeviceName: "blockdevice-10ad9f484c299597ed1e126d7b857967"
         poolConfig:
           dataRaidGroupType: "stripe"
   
       - nodeSelector:
           kubernetes.io/hostname: "worker-2" 
         dataRaidGroups:
           - blockDevices:
               - blockDeviceName: "blockdevice-3ec130dc1aa932eb4c5af1db4d73ea1b"
         poolConfig:
           dataRaidGroupType: "stripe"
      
       - nodeSelector:
           kubernetes.io/hostname: "worker-3"
         dataRaidGroups:
           - blockDevices:
               - blockDeviceName: "blockdevice-01afcdbe3a9c9e3b281c7133b2af1b68"
         poolConfig:
           dataRaidGroupType: "stripe"
   ```

2.  Apply the modified CSPC YAML.

    ```bash
    kubectl apply -f cspc-single.yaml
    ```
3. Check if the pool instances report their status as 'ONLINE'.

    ```bash
    kubectl get cspc -n openebs
    ```

    ```bash
    NAME            HEALTHYINSTANCES   PROVISIONEDINSTANCES   DESIREDINSTANCES   AGE
    cstor-storage   1                  1                      1                  2m2s

    ```

    ```bash
    kubectl get cspi -n openebs
    ```

    ```bash
    NAME                 HOSTNAME           ALLOCATED   FREE     CAPACITY   STATUS   AGE
    cstor-storage-vn92   worker1            260k        19900M   19900M     ONLINE   2m17s
    cstor-storage-al65   worker2            260k        19900M   19900M     ONLINE   2m17s
    cstor-storage-y7pn   worker3            260k        19900M   19900M     ONLINE   2m17s
    ```

4. Once your pool instances have come online, you can proceed with volume provisioning.
    Create a storageClass to dynamically provision volumes using OpenEBS CSI provisioner.
    A sample storageClass:

   ```yaml
   kind: StorageClass
   apiVersion: storage.k8s.io/v1
   metadata:
     name: cstor-csi
   provisioner: cstor.csi.openebs.io
   allowVolumeExpansion: true
   parameters:
     cas-type: cstor
     # cstorPoolCluster should have the name of the CSPC
     cstorPoolCluster: cstor-storage
     # replicaCount should be <= no. of CSPI
     replicaCount: "3"
   ```

   Create a storageClass using above example.

   ```bash
   kubectl apply -f csi-cstor-sc.yaml
   ```

   You will need to specify the correct cStor CSPC from your cluster
   and specify the desired `replicaCount` for the volume. The `replicaCount`
   should be less than or equal to the max pool instances available.

5. Create a PVC yaml using above created StorageClass name

    ```yaml
    kind: PersistentVolumeClaim
    apiVersion: v1
    metadata:
      name: demo-cstor-vol
    spec:
      storageClassName: cstor-csi
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 5Gi
     ```

    Apply the above pvc yaml to dynamically create volume and verify that
    the PVC has been successfully created and bound to a PersistentVolume (PV).

    ```bash
    $ kubectl get pvc
    NAME              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS       AGE
    demo-cstor-vol    Bound    pvc-52d88903-0518-11ea-b887-42010a80006c   5Gi        RWO            cstor-csi-stripe   10s
    ```

6. Verify that the all volume-specific resources have been created
    successfully. Check if CStorColumeConfig(cvc) is in `Bound` state.

    ```bash
    $ kubectl get cstorvolumeconfig -n openebs
    NAME                                         CAPACITY   STATUS    AGE
    pvc-52d88903-0518-11ea-b887-42010a80006c2    5Gi        Bound     60s
    ```

    Verify volume and its replicas are in `Healthy` state.

    ```bash
    $ kubectl get cstorvolume -n openebs
    NAME                                         CAPACITY   STATUS    AGE
    pvc-52d88903-0518-11ea-b887-42010a80006c2    5Gi        Healthy   60s
    ```

    ```bash
    $ kubectl get cstorvolumereplica -n openebs
    NAME                                                          ALLOCATED   USED    STATUS    AGE
    pvc-52d88903-0518-11ea-b887-42010a80006c-cstor-storage-vn92   6K          6K      Healthy   60s
    pvc-52d88903-0518-11ea-b887-42010a80006c-cstor-storage-al65   6K          6K      Healthy   60s
    pvc-52d88903-0518-11ea-b887-42010a80006c-cstor-storage-y7pn   6K          6K      Healthy   60s
    ```

7. Create an application and use the above created PVC.

    ```yaml
    apiVersion: v1
    kind: Pod
    metadata:
      name: busybox
      namespace: default
    spec:
      containers:
      - command:
           - sh
           - -c
           - 'date >> /mnt/openebs-csi/date.txt; hostname >> /mnt/openebs-csi/hostname.txt; sync; sleep 5; sync; tail -f /dev/null;'
        image: busybox
        imagePullPolicy: Always
        name: busybox
        volumeMounts:
        - mountPath: /mnt/openebs-csi
          name: demo-vol
      volumes:
      - name: demo-vol
        persistentVolumeClaim:
          claimName: demo-cstor-vol
    ```

    Verify that the pod is running and is able to write data to the volume.

    ```bash
    $ kubectl get pods
    NAME      READY   STATUS    RESTARTS   AGE
    busybox   1/1     Running   0          97s
    ```

    The example busybox application will write the current date into the
    mounted path at `/mnt/openebs-csi/date.txt` when it starts.

    ```bash
    $ kubectl exec -it busybox -- cat /mnt/openebs-csi/date.txt
    Wed Jul 12 07:00:26 UTC 2020
    ```
