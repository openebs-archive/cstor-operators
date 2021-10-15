# cStor Stripe Pool

## Single Pool Provisioning
Application of following YAML provisions a single node cStor pool.

**Note:** 

   i)Do not forget to modify the follwing CSPC YAML to add your hostname label of the k8s node.
   
   List the node to see the labels and modify accordingly.
   
   ```bash
   kubectl get node --show-labels
   ```
   List the block devices to add correct block device to the cspc yaml.
   
   ```bash
   kubernetes.io/hostname: "your-node"
   ```
   ii)Do not forget to modify the follwing CSPC YAML to add your blockdevice(block device should belong to the node where you want to provision).
   ```bash
         - blockDevices:
             - blockDeviceName: "your-block-device"
   ```

   The YAML looks like the following:
   
   ```yml
   apiVersion: cstor.openebs.io/v1
   kind: CStorPoolCluster
   metadata:
     name: cspc-stripe
     namespace: openebs
   spec:
     pools:
       - nodeSelector:
           kubernetes.io/hostname: "gke-cstor-demo-default-pool-3385ab41-2hkc"
         dataRaidGroups:
         - blockDevices:
             - blockDeviceName: "blockdevice-176cda34921fdae209bdd489fe72475d"
         poolConfig:
           dataRaidGroupType: "stripe"
   
   ```
### Steps
1. Apply the cspc yaml. ( Assuming that the file name is cspc.yaml that has the above content with modified blockdevice and node name)
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
    Please note that, a cspc can have mutiple cspi(s) so the cspc name prefix helps to figure out that
    the cspi belongs to which cspc.

3. You can have has many block devices you want in the cspc yaml.
   For example, see the following YAML.
   ```yml
   apiVersion: cstor.openebs.io/v1
   kind: CStorPoolCluster
   metadata:
     name: cspc-stripe
     namespace: openebs
   spec:
     pools:
       - nodeSelector:
           kubernetes.io/hostname: "worker-1"
         dataRaidGroups:
         - blockDevices:
             - blockDeviceName: "blockdevice-176cda34921fdae209bdd489fe72475d"
             - blockDeviceName: "blockdevice-176cda34921fdae209bdd489fe72475e"
             - blockDeviceName: "blockdevice-176cda34921fdae209bdd489fe72475f"
             - blockDeviceName: "blockdevice-176cda34921fdae209bdd489fe72475g"
         poolConfig:
           dataRaidGroupType: "stripe"
   ```
   The effective capacity of the pool here will be sum of all the capacities of the 4 block devices.
   In case you have provisioned a cspc stripe pool with only one block device, you can add block
   device later on also to expand the pool size.
   See [Pool Expansion by adding disk](#pool-expansion-by-adding-disk)
   
4. To delete the cStor pool
   
   ```bash
       kubectl delete cspc -n openebs cspc-stripe
   ```

## 3-Node Pool Provisioning
This tutorial is similar as that of above. We just need to add proper node and blockdevice names to our cspc yaml.

1. Following is a 3 node cspc stripe yaml configuration.
```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-stripe
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker-1"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-d1cd029f5ba4ada0db75adc8f6c88653"
      poolConfig:
        dataRaidGroupType: "stripe"

    - nodeSelector:
        kubernetes.io/hostname: "worker-2" 
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-79fe8245efd25bd7e7aabdf29eae4d71"
      poolConfig:
        dataRaidGroupType: "stripe"

    - nodeSelector:
        kubernetes.io/hostname: "worker-3"
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-50761ff2c406004e68ac2920e7215679"
      poolConfig:
        dataRaidGroupType: "stripe"
```

2. Apply the cspc yaml. ( Assuming that the file name is cspc.yaml that has the above content with modified blockdevice and node name)
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
        kubectl delete cspc -n openebs cspc-stripe
    ```

In case you have provisioned a cspc stripe pool on one node only, you can add more cStor pool instances for this cspc
on other nodes.
See [Pool Scale Up](pool-scale-up) 

## Pool Scale Up
Let us consider that following cspc yaml was applied to provision a cStor pool

```yml
   apiVersion: cstor.openebs.io/v1
   kind: CStorPoolCluster
   metadata:
     name: cspc-stripe-horizontal
     namespace: openebs
   spec:
     pools:
       - nodeSelector:
           kubernetes.io/hostname: "worker-1"
         dataRaidGroups:
         - blockDevices:
             - blockDeviceName: "blockdevice-45vda34921fdae209bdd489fe56775d"
         poolConfig:
           dataRaidGroupType: "stripe"
   
```
Run following command to see current pool instances(cspis)


```bash
    kubectl get cspi -n openebs
```
**Note:** If you have mutiple CSPC(s) created, there could be cspi(s) for that. 
          But note that, for this cspc you will have only one cspi as the CSPC
          spec says to provision cStor pool on one node only i.e. `worker-1`
          The cspi of this cspc will have a prefix `cspc-stripe-horizontal`
          in the name.


Now to create more pool instances for this cspc, just use `kubectl edit cspc -n openebs cspc-stripe-horizontal`
to add more node specs. Let us say, we want to add 2 more pool instances.

The YAML look like the following.
```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-stripe-horizontal
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker-1"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-45vda34921fdae209bdd489fe56775d"
      poolConfig:
        dataRaidGroupType: "stripe"
    # New node spec added -- to create a cStor pool on worker-2
    - nodeSelector:
        kubernetes.io/hostname: "worker-2" 
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-46vda6789fdae209bdd489fe56775d"
      poolConfig:
        dataRaidGroupType: "stripe"
    # New node spec added -- to create a cStor pool on worker-3
    - nodeSelector:
        kubernetes.io/hostname: "worker-3"
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-46vda0923fdae209bdd489fe56775d"
      poolConfig:
        dataRaidGroupType: "stripe"
```

## Pool Expansion By Adding Disk

Let us consider that following cspc yaml was applied to provision a cStor pool

```yml
   apiVersion: cstor.openebs.io/v1
   kind: CStorPoolCluster
   metadata:
     name: cspc-stripe-expand
     namespace: openebs
   spec:
     pools:
       - nodeSelector:
           kubernetes.io/hostname: "worker-1"
         dataRaidGroups:
         - blockDevices:
             - blockDeviceName: "blockdevice-45vda34921fdae209bdd489fe56775d"
         poolConfig:
           dataRaidGroupType: "stripe"
   
```
Run following command and keep in mind the pool size.

```bash
    kubectl get cspi -n openebs
```

Now to expand the pool on `worker-1` node, just use `kubectl edit cspc -n openebs cspc-stripe-expand` to add block devices for expansion
such that the YAML look like the following. You can add as many blockdevices for expansion as practical.
Just keep in mind that the new block device we are adding should belong to the same node.

```yml
   apiVersion: cstor.openebs.io/v1
   kind: CStorPoolCluster
   metadata:
     name: cspc-stripe-expand
     namespace: openebs
   spec:
     pools:
       - nodeSelector:
           kubernetes.io/hostname: "worker-1"
         dataRaidGroups:
         - blockDevices:
             - blockDeviceName: "blockdevice-45vda34921fdae209bdd489fe56775d"
             - blockDeviceName: "blockdevice-46vda34921fdae209bdd489fe56775d"
         poolConfig:
           dataRaidGroupType: "stripe"
   
```

Run following command and realise the increase in the pool size.

```bash
    kubectl get cspi -n openebs
```

You can also describe the cspi to see the events.

```bash
    kubectl describe cspi -n openebs <cspi-name>
```

```bash
Events:
  Type    Reason          Age   From               Message
  ----    ------          ----  ----               -------
  Normal  Created         69s   CStorPoolInstance  Pool created successfully
  Normal  Pool Expansion  14s   CStorPoolInstance  Pool Expanded Successfully By Adding BlockDevice Under Raid Group
```

Similarly, if we have a 3 node stripe pool, to expand pool on each node we can add blockdevice in the spec.
See the following example.
Let us consider that following cspc yaml was applied to provision cStor pool on 3 nodes.

```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-stripe-expand-multinode
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker-1"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-fg5649cf5ba4ada0db75adc8f6c88653"
      poolConfig:
        dataRaidGroupType: "stripe"

    - nodeSelector:
        kubernetes.io/hostname: "worker-2" 
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-gh785y65efd25bd7e7aabdf29eae4d71"
      poolConfig:
        dataRaidGroupType: "stripe"

    - nodeSelector:
        kubernetes.io/hostname: "worker-3"
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-9dfh6t52c406004e68ac2920e7215679"
      poolConfig:
        dataRaidGroupType: "stripe"
```

Run `kubectl edit cspc -n openebs cspc-stripe-expand-multinode`

Now,

i)   add `blockdevice-tt149cf5ba4ada0db75adc8f6c886544` to `worker-1` (blockdevice-tt149cf5ba4ada0db75adc8f6c88654 should belong to worker-1)

ii)  add `blockdevice-tt249cf5ba4ada0db75adc8f6c886544` to `worker-2` (blockdevice-tt249cf5ba4ada0db75adc8f6c88654 should belong to worker-2)

iii) add `blockdevice-tt349cf5ba4ada0db75adc8f6c886544` to `worker-3` (blockdevice-tt249cf5ba4ada0db75adc8f6c88654 should belong to worker-3)

The YAML looks like the following:
```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: cspc-stripe-expand-multinode
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: "worker-1"
      dataRaidGroups:
      - blockDevices:
          - blockDeviceName: "blockdevice-fg5649cf5ba4ada0db75adc8f6c88653"
          # New Block Device Added
          - blockDeviceName: "blockdevice-tt149cf5ba4ada0db75adc8f6c886544"
      poolConfig:
        dataRaidGroupType: "stripe"

    - nodeSelector:
        kubernetes.io/hostname: "worker-2" 
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-gh785y65efd25bd7e7aabdf29eae4d71"
            # New Block Device Added
            - blockDeviceName: "blockdevice-tt249cf5ba4ada0db75adc8f6c886544"
      poolConfig:
        dataRaidGroupType: "stripe"

    - nodeSelector:
        kubernetes.io/hostname: "worker-3"
      dataRaidGroups:
        - blockDevices:
            - blockDeviceName: "blockdevice-9dfh6t52c406004e68ac2920e7215679"
            # New Block Device Added
            - blockDeviceName: "blockdevice-tt349cf5ba4ada0db75adc8f6c886544"
      poolConfig:
        dataRaidGroupType: "stripe"
```

## Disk Replacement By Removing Disk
Stripe RAID configuration of cStor pool does not support disk replacement.
