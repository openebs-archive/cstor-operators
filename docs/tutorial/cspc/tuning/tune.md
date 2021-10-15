# Tuning cStor Pools

## Introduction 

CStor pool(s) can be tuned via CSPC and is the recommended way to do it. Following are list of tunables that can be applied:
- Resource requests and limits for pool manager containers to ensure quality of service.
- Toleration for pool manager pod to ensure scheduling of pool pods on tainted nodes. 
- Priority class for pool manager pod to specify priority levels as required. 
- Setting compression for cStor pools.
- Specifying read only threshold for cStor pools.

## Resource and Limits

Following CSPC YAML specifies `resources` and `auxResources` that will get applied to all pool manager pods for the CSPC.
`resources` gets applied to `cstor-pool` container and `auxResources` gets applied to side car containers i.e. `cstor-pool-mgmt` and `pool-exporter`.

In the following CSPC YAML we have only one pool spec (@spec.pools, notice that spec.pools is a list).
It is also possible to override the resource and limit value for a specific pool.
   
```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: demo-pool-cluster
  namespace: openebs
spec:
  resources:
    requests:
      memory: "2Gi"
      cpu: "250m"
    limits:
      memory: "4Gi"
      cpu: "500m"

  auxResources:
    requests:
      memory: "500Mi"
      cpu: "100m"
    limits:
      memory: "1Gi"
      cpu: "200m"
  pools:
    - nodeSelector:
        kubernetes.io/hostname: worker-node-1

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f36
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f37

      poolConfig:
        dataRaidGroupType: mirror
```

Following CSPC YAML explains how the resource and limits can be overridden.
If you look at the CSPC YAML, there is no `resources` and `auxResources` specified at pool level for `worker-node-1` and `worker-node-2` but specified for `worker-node-3`.
In this case, for `worker-node-1` and `worker-node-2` the `resources` and `auxResources` will be applied from @spec.resources and @spec.auxResources respectively but for `worker-node-3` these will be applied from @spec.pools[2].poolConfig.resources and @spec.pools[2].poolConfig.auxResources respectively.


```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: demo-pool-cluster
  namespace: openebs
spec:
  resources:
    requests:
      memory: "64Mi"
      cpu: "250m"
    limits:
      memory: "128Mi"
      cpu: "500m"

  auxResources:
    requests:
      memory: "50Mi"
      cpu: "400m"
    limits:
      memory: "100Mi"
      cpu: "400m"

  pools:
    - nodeSelector:
        kubernetes.io/hostname: worker-node-1

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f36
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f37

      poolConfig:
        dataRaidGroupType: mirror

    - nodeSelector:
        kubernetes.io/hostname: worker-node-2

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f39
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f40

      poolConfig:
        dataRaidGroupType: mirror

    - nodeSelector:
        kubernetes.io/hostname: worker-node-3

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f42
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f43

      poolConfig:
        dataRaidGroupType: mirror
        resources:
          requests:
            memory: 70Mi
            cpu: 300m
          limits:
            memory: 130Mi
            cpu: 600m

        auxResources:
          requests:
            memory: 60Mi
            cpu: 500m
          limits:
            memory: 120Mi
            cpu: 500m

```

## Toleration

Tolerations are applied in a similar manner like `resources` and `auxResources`.
The following is a sample CSPC YAML that has tolerations specified. For `worker-node-1` and `worker-node-2` tolerations are applied form @spec.tolerations but for `worker-node-3` it is applied from @spec.pools[2].poolConfig.tolerations
 
```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: demo-pool-cluster
  namespace: openebs
spec:

  tolerations:
  - key: data-plane-node
    operator: Equal
    value: true
    effect: NoSchedule

  pools:
    - nodeSelector:
        kubernetes.io/hostname: worker-node-1

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f36
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f37

      poolConfig:
        dataRaidGroupType: mirror

    - nodeSelector:
        kubernetes.io/hostname: worker-node-2

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f39
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f40

      poolConfig:
        dataRaidGroupType: mirror

    - nodeSelector:
        kubernetes.io/hostname: worker-node-3

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f42
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f43

      poolConfig:
        dataRaidGroupType: mirror
        tolerations:
        - key: data-plane-node
          operator: Equal
          value: true
          effect: NoSchedule

        - key: apac-zone
          operator: Equal
          value: true
          effect: NoSchedule
```

## Priority Class
Priority Class are also applied in a similar manner like `resources` and `auxResources`.
The following is a sample CSPC YAML that has priority class specified. For `worker-node-1` and `worker-node-2` priority class are applied form @spec.priorityClassName but for `worker-node-3` it is applied from @spec.pools[2].poolConfig.priorityClassName

Please visit this [link](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#priorityclass) for more information on priority class
 

*NOTE:* 
1. Priority class needs to be created before hand. In this case, `high-priority` and `ultra-priority` priority classes should exist.
2. The index starts form 0 for @.spec.pools list. 

```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: demo-pool-cluster
  namespace: openebs
spec:

  priorityClassName: high-priority 

  pools:
    - nodeSelector:
        kubernetes.io/hostname: worker-node-1

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f36
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f37

      poolConfig:
        dataRaidGroupType: mirror

    - nodeSelector:
        kubernetes.io/hostname: worker-node-2

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f39
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f40

      poolConfig:
        dataRaidGroupType: mirror

    - nodeSelector:
        kubernetes.io/hostname: worker-node-3

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f42
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f43

      poolConfig:
        dataRaidGroupType: mirror
        priorityClassName: utlra-priority

```

## Compression

Compression values can be set at pool level only. There is no override mechanism like it was there in case of `tolerations`, `resources`, `auxResources` and `priorityClass`.
Compression value must be one of `on`,`off`,`lzjb`,`gzip`,`gzip-[1-9]`,`zle` and `lz4`.

Note: lz4 is the default compression algorithm that is used if the compression field is left unspecified on the cspc.

```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: demo-pool-cluster
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: worker-node-1

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f36
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f37

      poolConfig:
        dataRaidGroupType: mirror
        compression: lz
```


## Read Only Threshold

RO threshold can be set in a similar manner like compression. ROThresholdLimit is threshold(percentage base) limit for pool read only mode. If ROThresholdLimit(%) amount of pool storage is consumed then pool will be set to readonly.
If ROThresholdLimit is set to 100 then entire pool storage will be used. By default it will be set to 85% i.e when unspecified on the CSPC.ROThresholdLimit value will be 0 < ROThresholdLimit <= 100.

```yml
apiVersion: cstor.openebs.io/v1
kind: CStorPoolCluster
metadata:
  name: demo-pool-cluster
  namespace: openebs
spec:
  pools:
    - nodeSelector:
        kubernetes.io/hostname: worker-node-1

      dataRaidGroups:
      - cspiBlockDevices:
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f36
          - blockDeviceName: blockdevice-ada8ef910929513c1ad650c08fbe3f37

      poolConfig:
        dataRaidGroupType: mirror

        roThresholdLimit : 70
```
