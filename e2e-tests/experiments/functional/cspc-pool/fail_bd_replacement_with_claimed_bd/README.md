## Experiment Metadata

Type  |     Description                                               | Storage    | K8s Platform | 
------|---------------------------------------------------------------|------------|--------------|
Functional | Verify the BlockDevice replacement is blocked by the admission server if the Bd is in claimed state | OpenEBS    | Any          | 

## Entry-Criteria

- cStor pool instances are in healthy state.
- Claimed BlockDevice should be available in the node to perform the disk replacement is blocked.

## Exit-Criteria

- cStor pool pods are should be Healthy.

### Application

Parameter     | Description
--------------|------------
POOL_NAME     | Name of the CSPC to perform the disk replacement.
OPERATOR_NS   | Namespace in Which OpenEBS components are deployed

## Procedure

- This e2e-test accepts the parameters in form of job environmental variables.
- This job patches the respective blockdevice can be replaced by the already inused blockdevice in a cspc pool and verifying the BD repalcement is blocked. 
