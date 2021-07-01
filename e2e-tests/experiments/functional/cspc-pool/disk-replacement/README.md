## Experiment Metadata

Type  |     Description                                               | Storage    | K8s Platform | 
------|---------------------------------------------------------------|------------|--------------|
Chaos |Replace the blockdevice/disk in cspc Pool and check the status | OpenEBS    | Any          | 

## Entry-Criteria

- Application services are accessible & pods are healthy
- Application writes are successful 
- cStor pool instances are in healthy state.
- Unclaimed BlockDevice should be available in the node to perform the disk replacement.

## Exit-Criteria

- Application services are accessible & pods are healthy
- Data written prior to chaos is successfully retrieved/read
- Database consistency is maintained as per db integrity check utils
- Storage target pods are healthy.
- cStor pool pods are should be Healthy.

### Application

Parameter     | Description
--------------|------------
APP_NAMESPACE | Namespace in which application pods are deployed
APP_LABEL     | Unique Labels in `key=value` format of application deployment
APP_PVC       | Name of persistent volume claim used for app's volume mounts 
POOL_NAME     | Name of the CSPC to perform the disk replacement.
OPERATOR_NS   | Namespace in Which OpenEBS components are deployed

### Health Checks 

Parameter             | Description
----------------------|------------
DATA_PERSISTENCY      | Data accessibility & integrity verification post recovery (enabled, disabled)


### Procedure

This scenario validates the behaviour of application and OpenEBS persistent volumes in the amidst of replace block device in the cStorPoolCluster  on OpenEBS data plane and control plane components.

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

The configmap data will be utilised by e2e experiments as its variables while executing the scenario. Based on the data provided, e2e-tests checks if the data is consistent after disk replacement is completed.

