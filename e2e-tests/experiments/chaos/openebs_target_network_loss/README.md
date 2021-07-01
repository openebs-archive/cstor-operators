## Experiment Metadata

Type  |     Description                                | Storage    |  Applications   | K8s Platform |     Tools       | 
------|------------------------------------------------|------------|-----------------|--------------|-----------------|
Chaos |Inject network loss on storage target/controller| OpenEBS    | Percona MySQL   | Any          | tc, netem       | 

## Entry-Criteria

- Application services are accessible & pods are healthy
- Application writes are successful 

## Exit-Criteria

- Application services are accessible & pods are healthy
- Data written prior to chaos is successfully retrieved/read
- Database consistency is maintained as per db integrity check utils
- Storage target pods are healthy

## Notes

- Typically used as a disruptive test, to cause loss of access to storage by injecting prolonged network loss
- Tests Recovery workflows for the PV & data integrity post recovery 

## Associated Utils 

- `inject_packet_loss_tc.yml`

## E2E experiment Environment Variables

### Application

Parameter     | Description
--------------|------------
APP_NAMESPACE | Namespace in which application pods are deployed
APP_LABEL     | Unique Labels in `key=value` format of application deployment
APP_PVC       | Name of persistent volume claim used for app's volume mounts 

### Health Checks 

Parameter             | Description
----------------------|------------
LIVENESS_APP_NAMESPACE| Namespace in which external liveness pods are deployed, if any
LIVENESS_APP_LABEL    | Unique Labels in `key=value` format for external liveness pod, if any
DATA_PERSISTENCY      | Data accessibility & integrity verification post recovery (enabled, disabled)

### Procedure

This scenario validates the behaviour of application and OpenEBS persistent volumes in the amidst of chaos induced on OpenEBS data plane and control plane components.

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

