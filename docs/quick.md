# Quickstart

### Prerequisites

Before setting up cStor operators make sure your Kubernetes Cluster
meets the following prerequisites:

1. You will need to have Kubernetes version 1.14 or higher
2. iSCSI initiator utils installed on all the worker nodes
3. You have access to install RBAC components into kube-system namespace.
4. You have disks attached to nodes to provision storage.

### Insallation

1.  Clone the repository.
    ```bash
    $ git clone https://github.com/openebs/cstor-operators.git
    ```
    
2.  Make sure you are at the root directory of the cloned repository.
    ```bash
    $ cd cstor-operators
    ```

3.  Apply RBAC.
    ```bash
    $ kubectl create -f deploy/rbac.yaml
    ```
4.  Install node disk manager(NDM).
    ```bash
    $ kubectl create -f deploy/ndm-operator.yaml
    ```
    Check that NDM daemonset and operator pods are running:

    ```bash
    $ kubectl get pod -n openebs
    NAME                                    READY   STATUS    RESTARTS   AGE
    openebs-ndm-f4kzc                       1/1     Running   0          37s
    openebs-ndm-operator-796f98fdd7-kvmpn   1/1     Running   0          37s
    openebs-ndm-t65b5                       1/1     Running   0          37s
    openebs-ndm-xztkj                       1/1     Running   0          37s
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

5.  Install cStor CRDs.
    ```bash
    $ kubectl create -f deploy/crds
    ```

6.  Install cStor operators.
    ```bash
    $ kubectl create -f deploy/cstor-operator.yaml
    ```
    Check that cspc-operator, cvc-operator and admission server pod has came up.

    ```bash
    $ kubectl get pods -n openebs
    NAME                                              READY   STATUS    RESTARTS   AGE
    cspc-operator-874cdcb6b-t4zcs                     1/1     Running   0          25s
    cvc-operator-fbcf99548-swlbg                      1/1     Running   0          25s
    openebs-cstor-admission-server-7c89777f8c-bbgwp   1/1     Running   0          25s
    openebs-ndm-f4kzc                                 1/1     Running   0          6m5s
    openebs-ndm-operator-796f98fdd7-kvmpn             1/1     Running   1          6m5s
    openebs-ndm-t65b5                                 1/1     Running   0          6m5s
    openebs-ndm-xztkj                                 1/1     Running   0          6m5s
    ```

7. Install CStor CSI driver.

   ```bash
    $ kubectl create -f deploy/csi-operator.yaml
   ```
    Check that cstor-csi-controller and cstor-csi-node pods has came up.

    ```bash
    $ kubectl get pods -n openebs
    NAME                                              READY   STATUS    RESTARTS   AGE
    cspc-operator-874cdcb6b-t4zcs                     1/1     Running   0          25s
    cvc-operator-fbcf99548-swlbg                      1/1     Running   0          25s
    openebs-cstor-admission-server-7c89777f8c-bbgwp   1/1     Running   0          25s
    openebs-ndm-f4kzc                                 1/1     Running   0          6m5s
    openebs-ndm-operator-796f98fdd7-kvmpn             1/1     Running   1          6m5s
    openebs-ndm-t65b5                                 1/1     Running   0          6m5s
    openebs-ndm-xztkj                                 1/1     Running   0          6m5s
    openebs-cstor-csi-controller-0                    7/7     Running   0          5m22s
    openebs-cstor-csi-node-86mx9                      2/2     Running   0          5m22s
    ```


8.  Provision a cStor pool. For simplicity, this guide will provision a 
    stripe pool on one node.
    Use the CSPC file from examples/cspc/cspc-single.yaml and modify by performing 
    follwing steps:
    
    i) Modify CSPC to add your node selector for the node where you want to provision the pool.
       List the nodes with labels:

       ```bash
       kubectl get node --show labels

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

       ```bash
       kubernetes.io/hostname: "worker1"
       ```

    i) Modify CSPC to add blockdevice attached to the same node where you want to provision the pool.
       ```bash
       kubectl get bd -n openebs
       ```
       ```bash
        NAME                                           NODENAME           SIZE          CLAIMSTATE   STATUS   AGE
        blockdevice-01afcdbe3a9c9e3b281c7133b2af1b68    worker3            21474836480   Unclaimed    Active   2m10s
        blockdevice-10ad9f484c299597ed1e126d7b857967    worker1            21474836480   Unclaimed    Active   2m17s
        blockdevice-3ec130dc1aa932eb4c5af1db4d73ea1b    worker2            21474836480   Unclaimed    Active   2m12s
       ```
       ```bash
       - blockDeviceName: "blockdevice-10ad9f484c299597ed1e126d7b857967"
       ```
       Finally the CSPC YAML looks like the following :
       ```yaml
       apiVersion: cstor.openebs.io/v1
       kind: CStorPoolCluster
       metadata:
         name: cspc-stripe
         namespace: openebs
       spec:
         pools:
           - nodeSelector:
               kubernetes.io/hostname: "worker1"
             dataRaidGroups:
             - blockDevices:
                 - blockDeviceName: "blockdevice-10ad9f484c299597ed1e126d7b857967" 
             poolConfig:
               dataRaidGroupType: "stripe"
       ```
9.  Apply the modified CSPC YAML.

    ```bash
    kubectl apply -f examples/cspc/cspc-single.yaml
    ```
10. Check if the pool has came online.
    
    ```bash
    kubectl get cspc -n openebs
    ```
   
    ```bash
    NAME          HEALTHYINSTANCES   PROVISIONEDINSTANCES   DESIREDINSTANCES   AGE
    cspc-stripe   1                  1                      1                  2m2s

    ```

    ```bash
    kubectl get cspi -n openebs
    ```

    ```bash
    NAME               HOSTNAME           ALLOCATED   FREE     CAPACITY   STATUS   AGE
    cspc-stripe-vn92   worker1            260k        19900M   19900M     ONLINE   2m17s
    ```
11. Once your pool has came online, we are ready with volume provisioning.
    Create a Storage Class to dynamically provision volumes using OpenEBS CSI provisioner.
    A sample storage class looks like:

   ```yaml
   kind: StorageClass
   apiVersion: storage.k8s.io/v1
   metadata:
     name: openebs-csi-cstor-stripe
   provisioner: cstor.csi.openebs.io
   allowVolumeExpansion: true
   parameters:
     cas-type: cstor
     cstorPoolCluster: cspc-stripe
     replicaCount: "1"
   ```

   Create StorageClass using above example

   ```bash
    $ kubectl apply -f csi-cstor-sc.yaml
   ```

   You will need to specify the correct cStor CSPC from your cluster
   and specify the desired `replicaCount` for the volume. The `replicaCount`
   should be less than or equal to the max pools available.

12. Create a PVC yaml with the StorageClass name

    ```yaml
    kind: PersistentVolumeClaim
    apiVersion: v1
    metadata:
      name: demo-cstor-vol
    spec:
      storageClassName: cstor-csi-stripe
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 5Gi
     ```

    Apply the above created pvc yaml to dynamically create volume and verify that
    the PVC has been successfully created and bound to a PersistentVolume(PV)

    ```bash
    $ kubectl get pvc
    NAME              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS       AGE
    demo-cstor-vol    Bound    pvc-52d88903-0518-11ea-b887-42010a80006c   5Gi        RWO            cstor-csi-stripe   10s
    ```

13. Verify that the all volume specific resources has been created
    successfully, check cstorvolumeconfig(cvc) should be in `Bound` state.

    ```bash
    $ kubectl get cstorvolumeconfig -n openebs
    NAME                                         CAPACITY   STATUS    AGE
    pvc-52d88903-0518-11ea-b887-42010a80006c2    5Gi        Bound   60s
    ```

    Verify volume and its replicas are `Healthy` state

    ```bash
    $ kubectl get cstorvolume -n openebs
    NAME                                         CAPACITY   STATUS    AGE
    pvc-52d88903-0518-11ea-b887-42010a80006c2    5Gi        Healthy   60s
    ```

    ```bash
    $ kubectl get cstorvolumereplica -n openebs
    NAME                                                        ALLOCATED   USED    STATUS    AGE
    pvc-52d88903-0518-11ea-b887-42010a80006c-cspc-stripe-vn92   6K          6K      Healthy   60s
    ```

13. Create an application and use the above created PVC

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

    Verify that the pods is running and is able to write the data.

    ```bash
    $ kubectl get pods
    NAME      READY   STATUS    RESTARTS   AGE
    busybox   1/1     Running   0          97s
    ```

    The example busybox application is instructed to write the date when it starts into the
    mounted path at `/mnt/openebs-csi/date.txt`

    ```bash
    $ kubectl exec -it busybox -- cat /mnt/openebs-csi/date.txt
    Wed Jul 12 07:00:26 UTC 2020
    ```
