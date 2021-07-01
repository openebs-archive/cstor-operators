## Experiment Metadata

| Type       | Description                                                  | Storage | Applications | K8s Platform |
| ---------- | ------------------------------------------------------------ | ------- | ------------ | ------------ |
| Functional | Increase the storage capacity of volume created using CSI. | OpenEBS| Any          | Any    |


## Supported platforms:

K8s : 1.18+

OS : Ubuntu, CentOS

## Entry-Criteria

- K8s nodes should be ready.
- cStor CSPC should be created in case of cstor volume resize.
- Application should be deployed using CSI provisioner.
- The feature-gate `--feature-gates=ExpandCSIVolumes` should be set to true.
- The module `iscsi_tcp` should be enabled.

## Exit-Criteria

- Volume should be resized successfully and application should be accessible seamlessly.
- Application should be able to use the new space.

## Notes

- This functional test checks if the csi volume can be resized.
- This e2e-test accepts the parameters in form of job environmental variables.
- This job updates the storage capacity of PVC with the new capacity and configures the PVC to increase its size.
- It checks if the application is accessible and the corresponding file system is scaled up.

## e2e-test Environment Variables

| Parameters    | Description                                            |
| ------------- | ------------------------------------------------------ |
| APP_NAMESPACE | Namespace where application and volume is deployed.    |
| APP_PVC       | Name of PVC whose storage capacity has to be increased |
| APP_LABEL     | Label of application pod                               |
| PV_CAPACITY   | Current storage capacity of volume                     |
| NEW_CAPACITY  | Desired storage capacity of volume                     |

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
kubectl logs -f <csi-volume-resize-xxxxx-xxxxx> -n e2e
```
To get the test-case result, get the corresponding e2e custom-resource `e2eresult` (short name: e2er ) and check its phase (Running or Completed) and result (Pass or Fail).

```
kubectl get e2er
kubectl get e2er csi-volume-resize -n e2e --no-headers -o custom-columns=:.spec.testStatus.phase
kubectl get e2er csi-volume-resize -n e2e --no-headers -o custom-columns=:.spec.testStatus.result
```