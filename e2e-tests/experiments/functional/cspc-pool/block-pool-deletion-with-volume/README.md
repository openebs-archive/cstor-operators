## About this experiment

Test Case to validate the deletion of cspc pool is blocked if the pool has volumes.

## Supported platforms:

K8s : 1.18+

OS : Ubuntu, CentOS

## Entry-criteria

- K8s cluster should be in healthy state including desired worker nodes in ready state.
- CSPC and CVC operators, openebs-ndm, cstor-csi-node daemonsets and csi controler statefulsets should be in running state.
- Application should be running successfully using cstor csi engine.
- cStor based storage pool should have been available.
- cStor CSI volumes should be available on the pool that needed to be delete.

## Exit-criteria

- CSPC pool deletion has to be blocked by the admission webhook server when the pool has the volume.

## Steps performed

- Obtain the CVR details from the selected application.
- From the cvr obtain the pool which has the volume.
- Delete the obtained pool from the cvr. Pool deletion has to fail with the error `volume still exists on the pool`
- Check the CVR and aplications are in Healthy state.

## How to run

- This experiment accepts the parameters in form of kubernetes job environmental variables.
- For running this experiment of csi volume resize, clone openens/cstor-operators[https://github.com/openebs/cstor-operators] repo and then first apply rbac and crds for e2e-framework.
```
kubectl apply -f cstor-operators/e2e-tests/hack/rbac.yaml
kubectl apply -f cstor-operators/e2e-tests/hack/crds.yaml
```
then update the needed test specific values in run_e2e_test.yml file and create the kubernetes job.
```
kubectl create -f run_e2e_test.yml
```
All the env variables description is provided with the comments in the same file.

After creating kubernetes job, when the job’s pod is instantiated, we can see the logs of that pod which is executing the test-case.

```
kubectl get pods -n e2e
kubectl logs -f <block-cspc-pool-deletion-if-pool-with-volume-xxxxx-xxxxx> -n e2e
```
To get the test-case result, get the corresponding e2e custom-resource `e2eresult` (short name: e2er ) and check its phase (Running or Completed) and result (Pass or Fail).

```
kubectl get e2er
kubectl get e2er block-cspc-pool-deletion-if-pool-with-volume -n e2e --no-headers -o custom-columns=:.spec.testStatus.phase
kubectl get e2er block-cspc-pool-deletion-if-pool-with-volume -n e2e --no-headers -o custom-columns=:.spec.testStatus.result
```