## Experiment Metadata

| Type       | Description                                           | Storage      | Applications | K8s Platform |
| ---------- | ----------------------------------------------------- | ------------ | ------------ | ------------ |
| Functional      | CSPC Pool expansion has to be blocked by admission server when the block device is already used by any pool | OpenBS cStor | Any          | Any          |

## Entry-Criteria

- K8s nodes should be ready.
- cStor CSPC should be created.
- Claimed BlockDevices should be available to expand the pool.

## Exit-Criteria

- Pool Expansion should be blocked by the Admission server.

## Procedure

- This test checks if the cspc pool Expansion can be Blocked successfully when the blockdevice already used by any other pool.
- This e2e-test accepts the parameters in form of job environmental variables.
- This job patches the respective CSPC pool to expand the cspc pool and verifying the pool expansion is blocked. 

## e2e-test Environment Variables

| Parameters    | Description                                            |
| ------------- | ------------------------------------------------------ |
| CSPC_NAME     | CSPC Pool name to expand                               |
| POOL_TYPE     | CSPC pool raid type [stripe,mirror,raidz,raidz2]       |
| OPERATOR_NS   | Nmaespace where the openebs is deployed                |
