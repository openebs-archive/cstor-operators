## About this experiment

 Test Case to validate the cspc pool creation is failed when the blockdevice has filesystem already on it.

## Supported platforms:

K8s : 1.18+

OS : Ubuntu

## Entry-criteria

- K8s cluster should be in healthy state including desired worker nodes in ready state.
- CSPC and CVC operators, openebs-ndm, cstor-csi-node daemonsets and csi controler statefulsets should be in running state.
- Application should be running successfully using cstor csi engine

## Exit-criteria

- CSPC pool creation has to be blocked by the admission webhook server.

## Steps performed

- Select the random node from the list of nodes.
- Obtain the unclaimed bockdevice from the selected node and get the device name (for ex /dev/sdb)
- SSH to the node and create the file system on the disk.
- Use the blockdevice to create the pool.
- Admission server will block the pool creation with the error `block device has file system`.
- Remove the file system created on the disk. 

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
kubectl logs -f <block-pool-creation-with-claimed-bd-xxxxx-xxxxx> -n e2e
```
To get the test-case result, get the corresponding e2e custom-resource `e2eresult` (short name: e2er ) and check its phase (Running or Completed) and result (Pass or Fail).

```
kubectl get e2er
kubectl get e2er block-pool-creation-with-claimed-bd -n e2e --no-headers -o custom-columns=:.spec.testStatus.phase
kubectl get e2er block-pool-creation-with-claimed-bd -n e2e --no-headers -o custom-columns=:.spec.testStatus.result
```