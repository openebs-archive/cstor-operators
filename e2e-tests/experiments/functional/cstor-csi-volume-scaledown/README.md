## Experiment Metadata

| Type       | Description                                           | Storage      | Applications | K8s Platform |
| ---------- | ----------------------------------------------------- | ------------ | ------------ | ------------ |
| Functional | Scale down the cStor volume replica of cstor-csi volume | OpenBS cStor | Any          | Any          |

## Entry-Criteria

- K8s nodes should be ready.
- cStor CSPC should be created.
- Openebs CSI provisioner should be running.
- Application should be deployed using with volume provisioned through CSI provisioner with multiple replicas.

## Exit-Criteria

- Volume replica should be scaled down and the dataset should be deleted from the corresponding pool..

## Procedure

- This functional test checks if the csi volume replicas can be scaled down.
- This e2e-test accepts the parameters in form of job environmental variables.
- This job patches the respective CVC by removing couple of pool instances from the manifest and checks if the scale down process is failing. Openebs admission server should restrict the scale down process here as it allows only one replica to be scaled down at once.
- This job verifies if the volume replicas can be scaled down one at a time.

## E2E-TEST Environment Variables

| Parameters    | Description                                            |
| ------------- | ------------------------------------------------------ |
| APP_NAMESPACE | Namespace where application and volume is deployed.    |
| APP_PVC       | Name of PVC whose storage capacity has to be increased |
| APP_LABEL     | Label of application pod                               |
| OPENEBS_NAMESPACE|Namespace where openebs components are deployed      |
|               |                                                        |
