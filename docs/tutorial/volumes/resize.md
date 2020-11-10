## How to Expand/Resize CStor Volume

OpenEBS Cstor introduces support for expanding an iSCSI PV using the CSI provisioner. Provided CStor is configured to function as a CSI provisioner, you can expand iSCSI PVs that have been created by CStor CSI Driver. This feature is supported with Kubernetes versions 1.16 and above.

For growing an CStor PV, you must ensure the following items are taken care of:

- The StorageClass must support volume expansion. This can be done by editing the StorageClass definition to set the `allowVolumeExpansion: true`.
- To resize a PV, edit the PVC definition and update the `spec.resources.requests.storage` to reflect the newly desired size, which must be greater than the original size.
- The PV must be attached to a pod for it to be resized. There are two scenarios when resizing an CStor PV:
    - If the PV is attached to a pod, CStor CSI driver expands the volume on the storage backend, rescans the device and resizes the filesystem.
    - When attempting to resize an unattached PV, CStor CSI driver expands the volume on the storage backend. Once the PVC is bound to a pod, driver rescans the device and resizes the filesystem. Kubernetes then updates the PVC size after the expand operation has successfully completed.

The example below shows how expanding CStor volumes works. For an already existing StorageClass, you can edit the StorageClass to include the `allowVolumeExpansion: true` parameter.

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: cstor-sparse-auto
provisioner: cstor.csi.openebs.io
allowVolumeExpansion: true
parameters:
  replicaCount: "3"
  cstorPoolCluster: "cspc-disk-pool"
  cas-type: "cstor"
```

For example a application `busybox` pod is using a below PVC associates with PV.

```sh
$ kubectl get pods
NAME            READY   STATUS    RESTARTS   AGE
busybox         1/1     Running   0          38m


$ kubectl get pvc
NAME                           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS              AGE
cstor-pvc                      Bound    pvc-849bd646-6d3f-4a87-909e-2416d4e00904   5Gi        RWO            cstor-csi-disk            1d

$ kubectl get pv
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                   STORAGECLASS        REASON   AGE
pvc-849bd646-6d3f-4a87-909e-2416d4e00904   5Gi        RWO            Delete           Bound    default/cstor-pvc       cstor-csi-disk               40m


```

To resize the PV that has been created from 5Gi to 10Gi, edit the PVC definition and update the `spec.resources.requests.storage` to 10Gi.
It may take few seconds to update the actual size in PVC resource, wait for the updated capacity to reflect in PVC status (pvc.status.capacity.storage).
It is internally a two step process for volumes containing a file system:
- Volume expansion
- FileSystem expansion


```sh
$ kubectl edit pvc cstor-pvc

apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  annotations:
    pv.kubernetes.io/bind-completed: "yes"
    pv.kubernetes.io/bound-by-controller: "yes"
    volume.beta.kubernetes.io/storage-provisioner: cstor.csi.openebs.io
  creationTimestamp: "2020-06-24T12:22:24Z"
  finalizers:
  - kubernetes.io/pvc-protection
    name: claim-csi-123
  namespace: default
  resourceVersion: "766"
  selfLink: /api/v1/namespaces/default/persistentvolumeclaims/claim-csi-123
  uid: 849bd646-6d3f-4a87-909e-2416d4e00904
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```

Now We can validate the resize has worked correctly by checking the size of the PVC, PV, or describing the pvc to get all events.

```sh
$ kubectl describe pvc cstor-pvc

Name:          claim-csi-123
Namespace:     default
StorageClass:  cstor-sparse-auto
Status:        Bound
Volume:        pvc-849bd646-6d3f-4a87-909e-2416d4e00904
Labels:        <none>
Annotations:   pv.kubernetes.io/bind-completed: yes
               pv.kubernetes.io/bound-by-controller: yes
               volume.beta.kubernetes.io/storage-provisioner: cstor.csi.openebs.io
Finalizers:    [kubernetes.io/pvc-protection]
Capacity:      10Gi
Access Modes:  RWO
VolumeMode:    Filesystem
Mounted By:    busybox-cstor
Events:
  Type     Reason                      Age                From                                                                                      Message
  ----     ------                      ----               ----                                                                                      -------
  Normal   ExternalProvisioning        46m (x2 over 46m)  persistentvolume-controller                                                               waiting for a volume to be created, either by external provisioner "cstor.csi.openebs.io" or manually created by system administrator
  Normal   Provisioning                46m                cstor.csi.openebs.io_openebs-cstor-csi-controller-0_bcba3893-c1c4-4e86-aee4-de98858ec0b7  External provisioner is provisioning volume for claim "default/claim-csi-123"
  Normal   ProvisioningSucceeded       46m                cstor.csi.openebs.io_openebs-cstor-csi-controller-0_bcba3893-c1c4-4e86-aee4-de98858ec0b7  Successfully provisioned volume pvc-849bd646-6d3f-4a87-909e-2416d4e00904
  Warning  ExternalExpanding           93s                volume_expand                                                                             Ignoring the PVC: didn't find a plugin capable of expanding the volume; waiting for an external controller to process this PVC.
  Normal   Resizing                    93s                external-resizer cstor.csi.openebs.io                                                     External resizer is resizing volume pvc-849bd646-6d3f-4a87-909e-2416d4e00904
  Normal   FileSystemResizeRequired    88s                external-resizer cstor.csi.openebs.io                                                     Require file system resize of volume on node
  Normal   FileSystemResizeSuccessful  4s                 kubelet, 127.0.0.1                                                                        MountVolume.NodeExpandVolume succeeded for volume "pvc-849bd646-6d3f-4a87-909e-2416d4e00904"

```

```sh
$ kubectl get pvc
NAME                           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS              AGE
cstor-pvc                      Bound    pvc-849bd646-6d3f-4a87-909e-2416d4e00904   10Gi        RWO            cstor-csi-disk            1d

$ kubectl get pv
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                   STORAGECLASS        REASON   AGE
pvc-849bd646-6d3f-4a87-909e-2416d4e00904   10Gi        RWO            Delete           Bound    default/cstor-pvc       cstor-csi-disk               40m
```
