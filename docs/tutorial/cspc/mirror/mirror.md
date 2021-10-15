# cStor mirror pool

## Single Node Pool Provisioning
Application of following YAML provisions a single node mirror cStor pool.

**Note:** 

   i)Do not forget to modify the following CSPC YAML to add your hostname label of the k8s node.
   
   List the node to see the labels and modify accordingly.
   
   ```bash
   kubectl get node --show-labels
   ```
   List the block devices to add correct block device to the cspc yaml.
   
   ```bash
   kubernetes.io/hostname: "your-node"
   ```
   ii)Do not forget to modify the following CSPC YAML to add your blockdevice(block device should belong to the node where you want to provision).
   ```bash
         - blockDevices:
             - blockDeviceName: "your-block-device-1"
             - blockDeviceName: "your-block-device-2"
   ```
   **Note:** You can add 2^n block devices in a raid group for mirror configuration.

   The YAML looks like the following:
   
   ```yml
   apiVersion: cstor.openebs.io/v1
   kind: CStorPoolCluster
   metadata:
     name: cspc-mirror-single
     namespace: openebs
   spec:
     pools:
       - nodeSelector:
           kubernetes.io/hostname: "worker-1"
         dataRaidGroups:
         - blockDevices:
             - blockDeviceName: "blockdevice-176cda34921fdae209bdd489fe72475d"
             - blockDeviceName: "blockdevice-189cda34sjdfjdf87dbdd489fe72475d"
         poolConfig:
           dataRaidGroupType: "mirror"
   
   ```
### Steps
1. Apply the cspc yaml. ( Assuming that the file name is cspc.yaml that has the above content with modified bd and node name)
    ```bash
        kubectl apply -f cspc.yaml
    ```
2. Run following commands to see the status.

    ```bash
        kubectl get cspc -n openebs
    ```

    ```bash
        kubectl get cspi -n openebs
    ```
    **Note:** 
    The name of cspi is prefixed by the cspc name indicating which cspc the cspi belongs to.
    Please note that, a cspc can have multiple cspi(s) so the cspc name prefix helps to figure out that
    the cspi belongs to which cspc.
   
3. To delete the cStor pool
   
   ```bash
       kubectl delete cspc -n openebs cspc-mirror
   ```

## 3-Node Pool Provisioning
This tutorial is similar as that of above. We just need to add proper node and bd names to our cspc yaml.

1. Following is a 3 node cspc mirror yaml configuration.
```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-mirror-multi
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker-1"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-d2cd029f5ba4ada0db75adc8f6c88654"
          - blockDeviceName: "blockdevice-d3cd029f5ba4ada0db75adc8f6c88655"
      poolConfig:
        dataRaidGroupType: "mirror"

    - nodeSelector:
        kubernetes.io/hostname: "worker-2" 
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-70fe8245efd25bd7e7aabdf29eae4d72"
            - blockDeviceName: "blockdevice-79fe8245efd25bd7e7aabdf29eae4d72"
      poolConfig:
        dataRaidGroupType: "mirror"

    - nodeSelector:
        kubernetes.io/hostname: "worker-3"
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-50761ff2c406004e68ac2920e7215670"
            - blockDeviceName: "blockdevice-51761ff2c406004e68ac2920e7215679"
      poolConfig:
        dataRaidGroupType: "mirror"
```

2. Apply the cspc yaml. ( Assuming that the file name is cspc.yaml that has the above content with modified bd and node name)
    ```bash
        kubectl apply -f cspc.yaml
    ```
3. Run following commands to see the status.

    ```bash
        kubectl get cspc -n openebs
    ```

    ```bash
        kubectl get cspi -n openebs
    ```
    **Note:** The name of cspi is prefixed by the cspc name that indicating which cspc the cspi belongs to.

4. To delete the cStor pool

    ```bash
        kubectl delete cspc -n openebs cspc-mirror
    ```

In case you have provisioned a cspc mirror pool on one node only, you can add more cStor pool instances for this cspc
on other nodes.
See [Pool Scale Up](pool-scale-up) 

## Pool Scale Up
Let us consider that following cspc yaml was applied to provision a cStor pool

```yml
   apiVersion: cstor.openebs.io/v1
   kind: CStorPoolCluster
   metadata:
     name: cspc-mirror-horizontal
     namespace: openebs
   spec:
     pools:
       - nodeSelector:
           kubernetes.io/hostname: "worker-1"
         dataRaidGroups:
         - blockDevices:
             - blockDeviceName: "blockdevice-45vda34921fdae209bdd489fe56775d"
             - blockDeviceName: "blockdevice-46vda34921fdae209bdd489fe56775e"
         poolConfig:
           dataRaidGroupType: "mirror"
   
```
Run following command to see current pool instances(cspis)


```bash
    kubectl get cspi -n openebs
```
**Note:** If you have mutiple CSPC(s) created, there could be cspi(s) for that. 
          But note that, for this cspc you will have only one cspi as the CSPC
          spec says to provision cStor pool on one node only i.e. `worker-1`
          The cspi of this cspc will have a prefix `cspc-mirror-horizontal`
          in the name.


Now to create more pool instances for this cspc, just use `kubectl edit cspc -n openebs cspc-mirror-horizontal`
to add more node specs. Let us say, we want to add 2 more pool instances.

The YAML look like the following.
```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-mirror-horizontal
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker-1"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-45vda34921fdae209bdd489fe56775d"
          - blockDeviceName: "blockdevice-46vda34921fdae209bdd489fe56775e"
      poolConfig:
        dataRaidGroupType: "mirror"
    # New node spec added -- to create a cStor pool on worker-2
    - nodeSelector:
        kubernetes.io/hostname: "worker-2" 
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-47vda6789fdae209bdd489fe56775c"
          - blockDeviceName: "blockdevice-48vda6789fdae209bdd489fe56775b"
      poolConfig:
        dataRaidGroupType: "mirror"
    # New node spec added -- to create a cStor pool on worker-3
    - nodeSelector:
        kubernetes.io/hostname: "worker-3"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-42vda0923fdae209bdd489fe56775a"
          - blockDeviceName: "blockdevice-43vda0923fdae209bdd489fe56775w"
      poolConfig:
        dataRaidGroupType: "mirror"
```

## Pool Expansion By Adding Disk

Let us consider that following cspc yaml was applied to provision a cStor pool

```yml
   apiVersion: cstor.openebs.io/v1
   kind: CStorPoolCluster
   metadata:
     name: cspc-mirror-expand
     namespace: openebs
   spec:
     pools:
       - nodeSelector:
           kubernetes.io/hostname: "worker-1"
         dataRaidGroups:
         - blockDevices:
             - blockDeviceName: "blockdevice-45vda34921fdae209bdd489fe56775a"
             - blockDeviceName: "blockdevice-46vda34921fdae209bdd489fe56775b"
         poolConfig:
           dataRaidGroupType: "mirror"
   
```
Run following command and keep in mind the pool size.

```bash
    kubectl get cspi -n openebs
```

Now to expand the pool on `worker-1` node, just use `kubectl edit cspc -n openebs cspc-mirror-expand` to add 
raid groups for expansion. Note that to expand pool of configuration other than stripe you have to add an entire
raid group.
The YAML look like the following.

```yml
   apiVersion: cstor.openebs.io/v1
   kind: CStorPoolCluster
   metadata:
     name: cspc-mirror-expand
     namespace: openebs
   spec:
     pools:
       - nodeSelector:
           kubernetes.io/hostname: "worker-1"
         dataRaidGroups:
         - blockDevices:
             - blockDeviceName: "blockdevice-45vda34921fdae209bdd489fe56775a"
             - blockDeviceName: "blockdevice-46vda34921fdae209bdd489fe56775b"
        # New Raid group added.
         - blockDevices:
             - blockDeviceName: "blockdevice-7ffce3846687bff3db128dd2872b0e46"
             - blockDeviceName: "blockdevice-3267fa8f5bede11e636cdf5e531bb265"
         poolConfig:
           dataRaidGroupType: "mirror"
   
```

If you describe the cspi you can see following events.
```bash
    kubectl describe cspi -n openebs <cspi-name>
```

```bash
Events:
  Type    Reason          Age    From               Message
  ----    ------          ----   ----               -------
  Normal  Created         2m14s  CStorPoolInstance  Pool created successfully
  Normal  Pool Expansion  23s    CStorPoolInstance  Pool Expanded Successfully By Adding RaidGroup With BlockDevices: [blockdevice-7ffce3846687bff3db128dd2872b0e46 blockdevice-3267fa8f5bede11e636cdf5e531bb265] device type: data pool type: mirror

```

Similarly, if we have a 3 node mirror pool, to expand pool on each node we can add raid groups in the spec.
See the following example.
Let us consider that following cspc yaml was applied to provision cStor pool on 3 nodes.

```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-mirror-expand-multinode
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker-1"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-fg5649cf5ba4ada0db75adc8f6c88653"
          - blockDeviceName: "blockdevice-ff5649cf5ba4ada0db75adc8f6c88654"
      poolConfig:
        dataRaidGroupType: "mirror"

    - nodeSelector:
        kubernetes.io/hostname: "worker-2" 
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-gh785y65efd25bd7e7aabdf29eae4d71"
            - blockDeviceName: "blockdevice-gh785y65efd25bd7e7aabdf29eae4d72"
      poolConfig:
        dataRaidGroupType: "mirror"

    - nodeSelector:
        kubernetes.io/hostname: "worker-3"
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-9efh6t52c406004e68ac2920e7215649"
            - blockDeviceName: "blockdevice-9ffh6t52c406004e68ac2920e7215659"
      poolConfig:
        dataRaidGroupType: "mirror"
```

Run `kubectl edit cspc -n openebs cspc-mirror-expand-multinode` to edit and expand.

The YAML looks like the following:
```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-mirror-expand-multinode
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker-1"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-fg5649cf5ba4ada0db75adc8f6c88653"
          - blockDeviceName: "blockdevice-ff5649cf5ba4ada0db75adc8f6c88654"
        # New Raid group added.
      - blockDevices:
          - blockDeviceName: "blockdevice-fg5649cf5ba4ada0db75adc8f6c88653"
          - blockDeviceName: "blockdevice-tt149cf5ba4ada0db75adc8f6c886544"
      poolConfig:
        dataRaidGroupType: "mirror"

    - nodeSelector:
        kubernetes.io/hostname: "worker-2" 
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-gh785y65efd25bd7e7aabdf29eae4d71"
            - blockDeviceName: "blockdevice-gh785y65efd25bd7e7aabdf29eae4d72"
        # New Raid group added.
        - blockDevices:
            - blockDeviceName: "blockdevice-gh785y65efd25bd7e7aabdf29eae4d71"
            - blockDeviceName: "blockdevice-tt249cf5ba4ada0db75adc8f6c886544"
      poolConfig:
        dataRaidGroupType: "mirror"

    - nodeSelector:
        kubernetes.io/hostname: "worker-3"
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-9efh6t52c406004e68ac2920e7215649"
            - blockDeviceName: "blockdevice-9ffh6t52c406004e68ac2920e7215659"
        # New Raid group added.
        - blockDevices:
            - blockDeviceName: "blockdevice-9dfh6t52c406004e68ac2920e7215679"
            - blockDeviceName: "blockdevice-tt349cf5ba4ada0db75adc8f6c886544"
      poolConfig:
        dataRaidGroupType: "mirror"
```

## Disk Replacement By Removing Disk
Stripe RAID configuration of cStor pool does not support disk replacement.

Let us consider following CSPC was provisioned.

```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-mirror-replace
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker1-ashutosh"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-10ad9f484c299597ed1e126d7b857967" 
          - blockDeviceName: "blockdevice-128f00677c03955784430f84def49736" 
      poolConfig:
        dataRaidGroupType: "mirror"
```

Now let us say, we want to replace `blockdevice-10ad9f484c299597ed1e126d7b857967` with `blockdevice-3267fa8f5bede11e636cdf5e531bb265`
The new block device `blockdevice-3267fa8f5bede11e636cdf5e531bb265` should belong to `worker-1` node and the capacity should
be greater than or equal to `blockdevice-10ad9f484c299597ed1e126d7b857967`

Run `kubectl edit cspc -n openebs cspc-mirror-replace` to edit the cspc

```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-mirror-replace
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker1-ashutosh"
      dataRaidGroups:
      - blockDevices:
          # The new block device blockdevice-3267fa8f5bede11e636cdf5e531bb265 placed.
          # And older one is removed. 
          - blockDeviceName: "blockdevice-3267fa8f5bede11e636cdf5e531bb265" 
          - blockDeviceName: "blockdevice-128f00677c03955784430f84def49736" 
      poolConfig:
        dataRaidGroupType: "mirror"
```

If you describe the cspi you will see following events

```bash
kubectl describe cspi -n openebs <cspi-name>
```


```bash
Events:
  Type    Reason                   Age   From               Message
  ----    ------                   ----  ----               -------
  Normal  Created                  74s   CStorPoolInstance  Pool created successfully
  Normal  BlockDevice Replacement  27s   CStorPoolInstance  Replacement of blockdevice-10ad9f484c299597ed1e126d7b857967 BlockDevice with blockdevice-3267fa8f5bede11e636cdf5e531bb265 BlockDevice is in-Progress
  Normal  BlockDevice Replacement  27s   CStorPoolInstance  Resilvering is successful on BlockDevice blockdevice-3267fa8f5bede11e636cdf5e531bb265

```
