# Application Pod Remains In ContainerCreating State Forever

cStor CSI drivers running on nodes are responsible for mounting/un-mounting the application volumes. As of now CSI driver purely acts based on gRPC request received to it(container orchestrator i.e kubelet will trigger appropriate RPC based on the pod event). When node went off abruptly for short time/having issues with kubelet or CSI node driver in this time when user **forcefully moved the application to schedule on new node** then issue can occur.

## How to identify mounting issue

There could be two mounting issues:

### case-1:
- Describe of application pod states `Volume pvc-ABCXYZ is still mounted on node machine-xyz`
  Further events on application pod:
  ```sh
  Events:
  Type     Reason       Age                   From                                   Message
  ----     ------       ----                  ----                                   -------
  Normal   Scheduled    <unknown>             default-scheduler                      Successfully assigned default/gitlab-postgresql-7d55d4bf85-vdp8l to gitlab-k8s-node3.mayalabs.io
  Warning  FailedMount  4m3s                  kubelet, gitlab-k8s-node3.mayalabs.io  Unable to attach or mount volumes: unmounted volumes=[data], unattached volumes=[data password-file default-token-v8vk5]: timed out waiting for the condition
  Warning  FailedMount  113s (x10 over 6m6s)  kubelet, gitlab-k8s-node3.mayalabs.io  MountVolume.MountDevice failed for volume "pvc-45dd46a9-1792-4d33-ad21-a851a16bb2b0" : rpc error: code = Internal desc = Volume pvc-45dd46a9-1792-4d33-ad21-a851a16bb2b0 still mounted on node gitlab-k8s-node4.mayalabs.io
  Warning  FailedMount  109s                  kubelet, gitlab-k8s-node3.mayalabs.io  Unable to attach or mount volumes: unmounted volumes=[data], unattached volumes=[default-token-v8vk5 data password-file]: timed out waiting for the condition
  ```

### case-2:
- Describe of application pod will have following iscsi error:
  ```sh
  Events:
  Type     Reason       Age   From               Message
  ----     ------       ----  ----               -------
  Normal   Scheduled    6s    default-scheduler  Successfully assigned default/fio-2-deployment-6c948d6799-kkbh2 to centos-worker-1
  Warning  FailedMount  1s    kubelet            MountVolume.MountDevice failed for volume "pvc-b29bf7ca-c569-4493-ac95-5cd6bb5fa2c3" : rpc error: code = Internal desc = failed to find device path: [], last error seen: failed to sendtargets to portal 10.101.20.108:3260, err: iscsiadm error: iscsiadm: Connection to Discovery Address 10.101.20.108 failediscsiadm: Login I/O error, failed to receive a PDUiscsiadm: retrying discovery login to 10.101.20.108iscsiadm: Connection to Discovery Address 10.101.20.108 failediscsiadm: Login I/O error, failed to receive a PDUiscsiadm: retrying discovery login to 10.101.20.108iscsiadm: Connection to Discovery Address 10.101.20.108 failediscsiadm: Login I/O error, failed to receive a PDUiscsiadm: retrying discovery login to 10.101.20.108iscsiadm: Connection to Discovery Address 10.101.20.108 failediscsiadm: Login I/O error, failed to receive a PDUiscsiadm: retrying discovery login to 10.101.20.108iscsiadm: Connection to Discovery Address 10.101.20.108 failediscsiadm: Login I/O error, failed to receive a PDUiscsiadm: retrying discovery login to 10.101.20.108iscsiadm: Connection to Discovery Address 10.101.20.108 failediscsiadm: Login I/O error, failed to receive a PDUiscsiadm: retrying discovery login to 10.101.20.108iscsiadm: connection login retries (reopen_max) 5 exceedediscsiadm: Could not perform SendTargets discovery: encountered iSCSI login failure (exit status 5)
  ```

# How to recover from this situation?

- [Case-1](#case-1) will happen when an application is forcibly moved to a different node manually by users/admins(It can be of any reason) when previous node is down. In this case, find out the corresponding CstorVolumeAttachemnt(CVA) of the volume and delete the CVA manually only if it is **eligible for deletion**:
  - Finding out the CVA resource for volume having an issue(CVA name will contains PV name as suffix)
    ```sh
    kubectl get cva -l Volname=pvc-b29bf7ca-c569-4493-ac95-5cd6bb5fa2c3 -n openebs
    ```
    Output:
    ```sh
    NAME                                                       AGE
    pvc-b29bf7ca-c569-4493-ac95-5cd6bb5fa2c3-centos-worker-1   9m45s
    ```
    Note: Replace above `pvc-b29bf7ca-c569-4493-ac95-5cd6bb5fa2c3` and `openebs` with corresponding PV name & OpenEBS namespace
  - If CVA pointing node is in `Ready` state and if there are no applications consuming volume from old node then CVA is eligible for deletion. Old node name can be found in labels by running the above command with `--show-labels` flag
    ```sh
      kubectl get cva -l Volname=pvc-b29bf7ca-c569-4493-ac95-5cd6bb5fa2c3 -n openebs --show-labels
    ```
    Output:
    ```sh
    NAME                                                       AGE   LABELS
    pvc-b29bf7ca-c569-4493-ac95-5cd6bb5fa2c3-centos-worker-1   17m   Volname=pvc-b29bf7ca-c569-4493-ac95-5cd6bb5fa2c3,nodeID=centos-worker-1
    ```
    Note: nodeID label value will be hostname where the volume is previouslly mounted.
  - Delete above identified CVA resource only if it is eligible
    ```sh
    kubectl delete cva pvc-b29bf7ca-c569-4493-ac95-5cd6bb5fa2c3-centos-worker-1 -n openebs
    ```

  Once above steps are executed successfully, application pod will be in `Running` state.

- In [case-2](#case-2) please reachout over [OpenEBS Channel](https://kubernetes.slack.com/messages/openebs/) in [K8s](https://kubernetes.slack.com) community.
