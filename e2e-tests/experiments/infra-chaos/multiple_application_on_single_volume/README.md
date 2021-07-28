## Experiment Metadata

| Type  | Description                                                  | Storage | K8s Platform      |
| ----- | ------------------------------------------------------------ | ------- | ----------------- |
| Chaos | Verify the application status when the multiple application pods trying to connect on single cstor volume | OpenEBS | on-premise-VMware |

## Entry-Criteria

- Application services are accessible & pods are healthy
- Application writes are successful

### Procedure

This scenario validates the behaviour of application and cStor persistent volumes in the amidst of multiple applications trying to connect the single cstor volume. steps followed in this test case
 - Verify the application that needs to perform chaos is in Healthy and Running state.
 - Dump data on the application to verify data availability post chaos. 
 - Obtain the application deployment and scale the application deployment.
 - Checking the Newly scaled pod is in `ContainerCreating` state and it's not mounted on the volume.
 - PowerOff the node where the application pod is in Running state. And Validated the pod went to `ContainerCreating` state and the scaled pod is in `Running` state.
 - Power on the node which shutdown during the chaos check the application pod is not login again. Verify the data availability in the application pod.
 - Checking the status of target pod and CVR's is in healthy state.

Based on the value of env `DATA_PERSISTENCE`, the corresponding data consistency util will be executed. At present, only busybox and percona-mysql are supported. Along with specifying env in the e2e experiment, user needs to pass name for configmap and the data consistency specific parameters required via configmap in the format as follows:

```
    parameters.yml: |
      blocksize: 4k
      blockcount: 1024
      testfile: difiletest
```

It is recommended to pass test-name for configmap and mount the corresponding configmap as volume in the e2e pod. The above snippet holds the parameters required for validation data consistency in busybox application.

For percona-mysql, the following parameters are to be injected into configmap.

```
    parameters.yml: |
      dbuser: root
      dbpassword: k8sDem0
      dbname: tdb
```

The configmap data will be utilised by e2e experiments as its variables while executing the scenario.

Based on the data provided, e2e checks if the data is consistent after recovering from induced chaos.

ESX password has to updated through k8s secret created. The e2e runner can retrieve the password from secret as environmental variable and utilize it for performing admin operations on the server.

Note: To perform admin operatons on vmware, the VM display name in hypervisor should match its hostname.

## Associated Utils

- `vm_power_operations.yml`,`mysql_data_persistence.yml`,`busybox_data_persistence.yml`

## e2e experiment Environment Variables

### Application

| Parameter        | Description                                                  |
| ---------------- | ------------------------------------------------------------ |
| APP_NAMESPACE    | Namespace in which application pods are deployed             |
| APP_LABEL        | Unique Labels in `key=value` format of application deployment |
| APP_PVC          | Name of persistent volume claim used for app's volume mounts |
| OPERATOR_NS      | Namespace where OpenEBS is installed                         |
| DATA_PERSISTENCE | Specify the application name against which data consistency has to be ensured. Example: busybox |

### Chaos

| Parameter    | Description                                                  |
| ------------ | ------------------------------------------------------------ |
| PLATFORM     | The platform where k8s cluster is created. Currently, only 'vmware' is supported. |
| ESX_HOST_IP  | The IP address of ESX server where the virtual machines are hosted. |
| ESX_PASSWORD | To be passed as configmap data.                              |
