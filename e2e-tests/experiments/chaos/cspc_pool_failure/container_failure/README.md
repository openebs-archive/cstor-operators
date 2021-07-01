## Experiment Metadata

| Type  | Description                                         | Storage | K8s Platform |
| ----- | --------------------------------------------------- | ------- | ------------ |
| Chaos | Kill the CSPC pool pod container and check if gets scheduled again | OpenEBS | Any          |

## Entry-Criteria

- Application services are accessible & pods are healthy
- Application writes are successful 

## Exit-Criteria

- Application services are accessible & pods are healthy
- Data written prior to chaos is successfully retrieved/read
- Database consistency is maintained as per db integrity check utils
- Storage target pods are healthy

## Notes

- Typically used as a disruptive test, to cause loss of access to storage pool by killing it.
- The pool pod should start again and it should be healthy.

## Associated Utils 

- `cstor_cspc_pool_kill.yml`,`pod_failure_by_sigkill.yaml`

### Procedure

This scenario validates the behaviour of application and OpenEBS persistent volumes in the amidst of chaos induced on storage pool. The e2e experiment fails the specified pool and thereby losing the access to volumes being created on it.

After injecting the chaos into the component specified via environmental variable, e2e experiment observes the behaviour of corresponding OpenEBS PV and the application which consumes the volume.

Based on the value of env DATA_PERSISTENCE, the corresponding data consistency util will be executed. At present only busybox and percona-mysql are supported. Along with specifying env in the e2e experiment, user needs to pass name for configmap and the data consistency specific parameters required via configmap in the format as follows:

    parameters.yml: |
      blocksize: 4k
      blockcount: 1024
      testfile: difiletest

It is recommended to pass test-name for configmap and mount the corresponding configmap as volume in the e2e pod. The above snippet holds the parameters required for validation data consistency in busybox application.

For percona-mysql, the following parameters are to be injected into configmap.

    parameters.yml: |
      dbuser: root
      dbpassword: k8sDem0
      dbname: tdb

The configmap data will be utilised by e2e experiments as its variables while executing the scenario. Based on the data provided, e2e checks if the data is consistent after recovering from induced chaos.

## e2e-test Environment Variables

### Application

| Parameter     | Description                                                  |
| ------------- | ------------------------------------------------------------ |
| APP_NAMESPACE | Namespace in which application pods are deployed             |
| APP_LABEL     | Unique Labels in `key=value` format of application deployment |
| APP_PVC       | Name of persistent volume claim used for app's volume mounts |

### chaos parameters

| Parameter     | Description                                                  |
| ------------- | ------------------------------------------------------------ |
| CONTAINER_RUNTIME | Type of container run time used (docker,containerd,cri-o)|
| CONTAINER_NAME    | Name of the container to perform the chaos (cstor-pool or cstor-pool-mgmt) |

### Health Checks 

| Parameter              | Description                                                  |
| ---------------------- | ------------------------------------------------------------ |
| DATA_PERSISTENCE       | Data accessibility & integrity verification post recovery. To check against busybox set value: "busybox" and for percona, set value: "mysql"|
