# Block Device Tagging

NDM provides a feature to reserve block devices to be used for specific 
applications via block device tags. This feature can also be used by cStor 
operators to specify the block devices that should be consumed by cstor pools 
and conversely restrict anyone else from using those block devices. This feature
can help in protecting against manual errors in specifying the block devices in 
the CSPC yamls by users.

## How to use it ?

This tutorial will walk through the steps to tag a block device and allow it to
be used by only a specific CSPC. 

The prerequisites for this tutorial is : 
- You should have a basic understading of cStor CSPC in general.
- Follow this [link](../../../../docs/quick.md) and install cstor-operators. 


Consider the following blockdevice in a Kubernetes cluster which will be tried
to use to provision a storage pool.

```bash
$ kubectl get bd -n openebs --show-labels
NAME                                           NODENAME           SIZE          CLAIMSTATE   STATUS   AGE   LABELS
blockdevice-00439dc464b785256242113bf0ef64b9   worker3   21473771008   Unclaimed    Active   34h   kubernetes.io/hostname=worker3,ndm.io/blockdevice-type=blockdevice,ndm.io/managed=true
blockdevice-022674b5f97f06195fe962a7a61fcb64   worker1   21473771008   Unclaimed    Active   34h   kubernetes.io/hostname=worker1,ndm.io/blockdevice-type=blockdevice,ndm.io/managed=true

blockdevice-241fb162b8d0eafc640ed89588a832df   worker2   21473771008   Unclaimed    Active   34h   kubernetes.io/hostname=worker2,ndm.io/blockdevice-type=blockdevice,ndm.io/managed=true
```

Now, tag the block device using openebs.io/block-device-tag label.

```bash
# Tagging block device blockdevice-00439dc464b785256242113bf0ef64b9 only
$ kubectl label bd blockdevice-00439dc464b785256242113bf0ef64b9 -n openebs  openebs.io/block-device-tag=fast
blockdevice.openebs.io/blockdevice-00439dc464b785256242113bf0ef64b9 labeled

$ kubectl get bd -n openebs blockdevice-00439dc464b785256242113bf0ef64b9 --show-labels
NAME                                           NODENAME           SIZE          CLAIMSTATE   STATUS   AGE   LABELS
blockdevice-00439dc464b785256242113bf0ef64b9   worker3-ashutosh   21473771008   Unclaimed    Active   34h   kubernetes.io/hostname=worker3-ashutosh,ndm.io/blockdevice-type=blockdevice,ndm.io/managed=true,openebs.io/block-device-tag=fast
```
 
Following is the CSPC that will used to provision cStor pools:

```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-stripe
  namespace: openebs
  annotations:
   # This annotaion help specify the BD that can be allowed. 
   openebs.io/allowed-bd-tags: cstor,ssd
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker1-ashutosh"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-022674b5f97f06195fe962a7a61fcb64"
      poolConfig:
        dataRaidGroupType: "stripe"
- nodeSelector:
        kubernetes.io/hostname: "worker2-ashutosh"
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-241fb162b8d0eafc640ed89588a832df"
      poolConfig:
        dataRaidGroupType: "stripe"
- nodeSelector:
        kubernetes.io/hostname: "worker3-ashutosh"
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-00439dc464b785256242113bf0ef64b9"
      poolConfig:
        dataRaidGroupType: "stripe"
```

Do a `kubectl apply -f` of the above CSPC file:

```bash
$ kubectl apply -f cspc.yaml 
cstorpoolcluster.cstor.openebs.io/cspc-stripe created
$ kubectl get cspi -n openebs
NAME               HOSTNAME           FREE     CAPACITY    READONLY   PROVISIONEDREPLICAS   HEALTHYREPLICAS   STATUS   AGE
cspc-stripe-b9f6   worker2   19300M   19300614k   false      0                     0                 ONLINE   89s
cspc-stripe-q7xn   worker1   19300M   19300614k   false      0                     0                 ONLINE   89s
```

You can see that CSPI for node `worker3` is not created and the reason for this
is the following:
- CSPC YAML created above has `openebs.io/allowed-bd-tags: cstor,ssd` in its 
annotation. This means that the CSPC operator will only consider those block
devices for provisioning that do not have a BD tag `openebs.io/block-device-tag` 
on the block device or has the tag but the values are either `cstor` or `ssd`.

- In this case the `blockdevice-022674b5f97f06195fe962a7a61fcb64` 
(on node `worker1`) and `blockdevice-241fb162b8d0eafc640ed89588a832df`
(on node `worker2`) does not have a tag with key `openebs.io/block-device-tag`.
Hence, no restriction is applied on it and it can be used the the CSPC operator
for pool provisioning or operations.

- In the case of blockdevice `blockdevice-00439dc464b785256242113bf0ef64b9`
(on node `worker3`) it has a block device tag `openebs.io/block-device-tag` with
value `fast`. (This tag is simply a Kubernetes label). But on the CSPC, the
annotation `openebs.io/allowed-bd-tags` has value `cstor` and `ssd`. There is no
`fast` keyword present in the annotation value and hence this BD cannot be used. 

Now edit this CSPC to add `fast` in the annotation `openebs.io/allowed-bd-tags`
via Kubectl. 

```yml
openebs.io/allowed-bd-tags: cstor,ssd,fast

```
After adding the annotation on the CSPC the CSPI will get created.

```bash
$ kubectl get cspi -n openebs
NAME               HOSTNAME           FREE     CAPACITY    READONLY   PROVISIONEDREPLICAS   HEALTHYREPLICAS   STATUS   AGE
cspc-stripe-b9f6   worker2   19300M   19300074k   false      0                     0                 ONLINE   7m8s
cspc-stripe-lznh   worker3   19300M   19300053k   false      0                     0                 ONLINE   5s
cspc-stripe-q7xn   worker1   19300M   19300074k   false      0                     0                 ONLINE   7m8s
```

## More Info
- If you want BD of multiple tag values to be allowed, the value for allowed bd 
tag annotation can be written in the following comma-separated manner:
```yml
openebs.io/allowed-bd-tags: fast,ssd,nvme
```
- A BD tag has only one value on the block device CR. For example 
    - `openebs.io/block-device-tag: fast` — fast is the value.
    - `openebs.io/block-device-tag: fast,ssd` — block devices should not be tagged 
    in this format. One of the reasons for this is, cStor allowed bd tag annotation 
    takes comma-separated values and values like above(i.e `fast,ssd` ) can never 
    be interpreted as a single word in cStor and hence BDs tagged in above format cannot 
    be utilised by cStor.

- If any block device mentioned in CSPC has an empty value for the 
`openebs.io/block-device-tag` then it will not be considered for pool
provisioning and operations. Block devices with empty tag value are implicitly 
not allowed by the CSPC operator.