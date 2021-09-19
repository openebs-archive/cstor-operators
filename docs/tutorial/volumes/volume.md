### Prerequisites

Before setting up OpenEBS CStor CSI driver make sure your Kubernetes Cluster
meets the following prerequisites:

1. You will need to have Kubernetes version 1.16 or higher
2. You will need to have CStor-Operator installed.
   The steps to install cstor operators are [here](../../deploy/cstor-operators)
3. CStor CSI driver operates on the cStor Pools provisioned using the new schema called CSPC.
   Steps to provision the pools using the same are [here](./../intro.md)
4. iSCSI initiator utils installed on all the worker nodes
5. You have access to install RBAC components into kube-system namespace.
   The OpenEBS CStor CSI driver components are installed in kube-system
   namespace to allow them to be flagged as system critical components.

### Setup OpenEBS CStor CSI Driver

OpenEBS CStor CSI driver comprises of 2 components:
- A controller component launched as a StatefulSet,
  implementing the CSI controller services. The Control Plane
  services are responsible for creating/deleting the required
  OpenEBS Volume.
- A node component that runs as a DaemonSet,
  implementing the CSI node services. The node component is
  responsible for performing the iSCSI connection management and
  connecting to the OpenEBS Volume.

OpenEBS CStor CSI driver components can be installed by running the
following command.

The node components make use of the host iSCSI binaries for iSCSI
connection management. Depending on the OS, the spec will have to
be modified to load the required iSCSI files into the node pods.

Depending on the OS select the appropriate deployment file.

- For Ubuntu 16.04 and CentOS.
  ```
  kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-csi/HEAD/deploy/csi-operator.yaml
  ```

- For Ubuntu 18.04
  ```
  kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-csi/HEAD/deploy/csi-operator-ubuntu-18.04.yaml
  ```

Verify that the OpenEBS CSI Components are installed.

```
$ kubectl get pods -n kube-system -l role=openebs-csi
NAME                       READY   STATUS    RESTARTS   AGE
openebs-csi-controller-0   4/4     Running   0          6m14s
openebs-csi-node-56t5g     2/2     Running   0          6m13s

```


### Provision a cStor volume

1. Make sure you already have a cStor Pool Created or you can
   create one using the below command. In the below cspc.yaml make sure
   that the specified pools list should be greater than or equal to
   the number of replicas required for the volume. Update `kubernetes.io/hostname`
   and `blockDeviceName` in the below yaml before applying the same.

   The following command will create the specified cStor Pools in the cspc yaml:

   ```
   kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-csi/HEAD/examples/cspc.yaml
   ```

2. Create a Storage Class to dynamically provision volumes
   using OpenEBS CSI provisioner. A sample storage class looks like:
   ```
   kind: StorageClass
   apiVersion: storage.k8s.io/v1
   metadata:
     name: openebs-csi-cstor-sparse
   provisioner: cstor.csi.openebs.io
   allowVolumeExpansion: true
   parameters:
     cas-type: cstor
     cstorPoolCluster: cstor-sparse-cspc
     replicaCount: "1"
   ```
   You will need to specify the correct cStor CSPC from your cluster
   and specify the desired `replicaCount` for the volume. The `replicaCount`
   should be less than or equal to the max pools available.

   The following file helps you to create a Storage Class
   using the cStor sparse pool created in the previous step.
   ```
   kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-csi/HEAD/examples/csi-storageclass.yaml
   ```

3. Run your application by specifying the above Storage Class for
   the PVCs.

   The following example launches a busybox pod using a cStor Volume
   provisioned via CSI Provisioner.
   ```
   kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-csi/HEAD/examples/busybox-csi-cstor-sparse.yaml
   ```

   Verify that the pods is running and is able to write the data.
   ```
   $ kubectl get pods
   NAME      READY   STATUS    RESTARTS   AGE
   busybox   1/1     Running   0          97s
   ```

   The busybox is instructed to write the date when it starts into the
   mounted path at `/mnt/openebs-csi/date.txt`

   ```
   $ kubectl exec -it busybox -- cat /mnt/openebs-csi/date.txt
   Wed Jul 31 04:56:26 UTC 2019
   ```
