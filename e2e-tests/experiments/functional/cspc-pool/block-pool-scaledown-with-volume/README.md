## Experiment Metadata

| Type  | Description                                                  | Storage | K8s Platform |
| ----- | ------------------------------------------------------------ | ------- | ------------ |
| Day-2-ops |  Scale down the cstor CSPC pool and check if it scaled down successfully | OpenEBS | Any          |

#### Description

The goal of this test is to scale down the cstor CSPC pool and check if its successfully scaled down.

#### Prerequisites

- cStor based storage pool should have been available.
- cStor CSI volumes shouldn't be available on the pool that needed to be scale down.

#### e2e experiment Environment Variables

| Parameter        | Description                                                  |
| ---------------- | ------------------------------------------------------------ |
| POOL_NAME        | Names of CSPC pool name to scaledown the cspi pools          |
| OPERATOR_NS      | Namespace where OpenEBS is installed                         |


#### Procedure

- Verify the CSPI are all in `ONLINE` state and  Check the Pool pods are all running
- Select the pool [cspi] want to scale down.
- Obtain the node name of the targeted CSPI for scaledown.
- Remove the CSPI entry from the CSPC 

#### Expected result

- blockDevices of the scaled down CSPI should be in Unclaimed state.
- CSPI and the pool pod corresponding to scaled down pool should be deleted

