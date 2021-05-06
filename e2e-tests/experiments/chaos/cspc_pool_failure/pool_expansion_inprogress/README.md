## Experiment Metadata

| Type       | Description                                           | Storage      | Applications | K8s Platform |
| ---------- | ----------------------------------------------------- | ------------ | ------------ | ------------ |
| CHAOS      | Restart the pool pod while CSPC pool expansion is in progress | OpenBS cStor | Any          | Any          |

## Entry-Criteria

- K8s nodes should be ready.
- cStor CSPC should be created.
- Application should be deployed using volume provisioned through CSI provisioner.
- Unclaimed BlockDevices should be available to expand the pool.

## Exit-Criteria

- Pool should be expanded successfully and the application should be still up and running without any disruption.

## Procedure

- This chaos test checks if the cspc pool can be expanded successfully even if the respective pool pod is restarted.
- This e2e-test accepts the parameters in form of job environmental variables.
- This job patches the respective CSPC pool to expand the cspc pool and verifying the status of cspi. 

## E2E-test Environment Variables

| Parameters    | Description                                            |
| ------------- | ------------------------------------------------------ |
| CSPC_NAME     | CSPC Pool name to expand                               |
| POOL_TYPE     | CSPC pool raid type [stripe,mirror,raidz,raidz2]       |
| OPERATOR_NS   | Nmaespace where the openebs is deployed                |
