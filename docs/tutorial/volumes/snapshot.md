## How to Create Volume Snapshots and Restore Volumes from Snapshots in Kubernetes Clusters


You must have an existing volume in use in your cluster, which you can create by creating a PersistentVolumeClaim (PVC). For the purposes of this tutorial, presume we have already created a PVC by calling `kubectl create -f your_pvc_file.yaml` 
with a YAML file that looks like this:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: cstor-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
  storageClassName: cstor-csi-disk
```


The example detailed below explains the constructs required for working with snapshots and shows how snapshots can be created and used.
Before creating a Volume Snapshot, a `VolumeSnapshotClass` must be set up.

```yaml
kind: VolumeSnapshotClass
apiVersion: snapshot.storage.k8s.io/v1
metadata:
  name: csi-cstor-snapshotclass
  annotations:
    snapshot.storage.kubernetes.io/is-default-class: "true"
driver: cstor.csi.openebs.io
deletionPolicy: Delete
```

The driver points to OpenEBS CStor CSI driver. The `deletionPolicy` can be set to `Delete` or `Retain`. When set to Retain, the underlying physical snapshot on the storage cluster is retained even when the VolumeSnapshot object is deleted.

#### Note: In some cluster like OpenShift(OCP) 4.5, which only installs the `v1beta1` version of `VolumeSnapshotClass` as supported version, then you may get the API error like:

```
$ kubectl apply -f snapshotclass.yaml
no matches for kind "VolumeSnapshotClass" in version "snapshot.storage.k8s.io/v1"
```
in such cases you can change the apiVersion to use `v1beta1` version instead of `v1` shown below:

```yaml
kind: VolumeSnapshotClass
apiVersion: snapshot.storage.k8s.io/v1beta1
metadata:
  name: csi-cstor-snapshotclass
  annotations:
    snapshot.storage.kubernetes.io/is-default-class: "true"
driver: cstor.csi.openebs.io
deletionPolicy: Delete
```

### Create a Snapshot of a Volume

To create a snapshot of a volume, here's an example of a YAML file that defines a snapshot:

```yaml
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: cstor-pvc-snap
spec:
  volumeSnapshotClassName: csi-cstor-snapshotclass
  source:
    persistentVolumeClaimName: cstor-pvc
```

The snapshot is being created for a PVC named `cstor-pvc`, and the name of the snapshot is set to `cstor-pvc-snap`.

```sh
$ kubectl create -f snapshot.yaml
volumesnapshot.snapshot.storage.k8s.io/cstor-pvc-snap created

$ kubectl get volumesnapshots
NAME                   AGE
cstor-pvc-snap              10s
```

This created a `VolumeSnapshot` object. A VolumeSnapshot is analogous to a PVC and is associated with a `VolumeSnapshotContent` object that represents the actual snapshot.
To identify the `VolumeSnapshotContent` object for the `cstor-pvc-snap` VolumeSnapshot by describing it.

```
$ kubectl describe volumesnapshots cstor-pvc-snap
Name:         cstor-pvc-snap
Namespace:    default
.
.
.
Spec:
  Snapshot Class Name:    cstor-csi-snapshotclass
  Snapshot Content Name:  snapcontent-e8d8a0ca-9826-11e9-9807-525400f3f660
  Source:
    API Group:
    Kind:       PersistentVolumeClaim
    Name:       cstor-pvc
Status:
  Creation Time:  2020-06-20T15:27:29Z
  Ready To Use:   true
  Restore Size:   5Gi
.
.
```


The `SnapshotContentName` identifies the `VolumeSnapshotContent` object which serves this snapshot. The Ready To Use parameter indicates that the Snapshot created successfully and 
can be used to create a new PVC.

#### Note: In some cluster like OpenShift(OCP) 4.5, which only installs the `v1beta1` version of `VolumeSnapshots` as supported version, then you may get the API error like:

```
$ kubectl apply -f snapshot.yaml
no matches for kind "VolumeSnapshot" in version "snapshot.storage.k8s.io/v1"
```
in such cases you can change the apiVersion to use `v1beta1` version instead of `v1` shown below:

```yaml
apiVersion: snapshot.storage.k8s.io/v1beta1
kind: VolumeSnapshot
metadata:
  name: cstor-pvc-snap
spec:
  volumeSnapshotClassName: csi-cstor-snapshotclass
  source:
    persistentVolumeClaimName: cstor-pvc
```

### Create PVCs from VolumeSnapshots

To restore from a given snapshot, you need to create a new PVC that refers to the snapshot.Here's an example of a YAML file that restores from a snapshot and creates a new PVC:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: restore-cstor-pvc
spec:
  storageClassName: cstor-csi-disk
  dataSource:
    name: cstor-pvc-snap
    kind: VolumeSnapshot
    apiGroup: snapshot.storage.k8s.io
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```

The `dataSource` shows that the PVC must be created using a `VolumeSnapshot` named `cstor-pvc-snap` as the source of the data. This instructs CStor CSI to create a PVC from the snapshot. Once the PVC is created, it can be attached to a pod and used just like any other PVC.


5. Verify that the PVC has been successfully created:

```
kubectl get pvc
NAME                           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS              AGE
cstor-pvc                      Bound    pvc-52d88903-0518-11ea-b887-42010a80006c   5Gi        RWO            cstor-csi-disk            1d
restore-cstor-pvc              Bound    pvc-2f2d65fc-0784-11ea-b887-42010a80006c   5Gi        RWO            cstor-csi-disk            5s
```
