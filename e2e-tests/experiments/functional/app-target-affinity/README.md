## About this experiment

This experiment verifies the storage volume by connecting to the Volume Target Pod. Target Pod Affinity policy can be used to co-locate volume target pod on the same node as workload.

Configured Policy having target-affinity label for example, using kubernetes.io/hostname as a topologyKey in CStorVolumePolicy
for e.g.
```
apiVersion: cstor.openebs.io/v1
kind: CStorVolumePolicy
metadata:
  name: csi-volume-policy
  namespace: openebs
spec:
  target:
    affinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: openebs.io/target-affinity
            operator: In
            values:
            - app-label                              // application-unique-label
        topologyKey: kubernetes.io/hostname
        namespaces: ["default"]                      // application namespace
```

## Supported platforms:

K8s : 1.18+

OS : Ubuntu, CentOS

## Entry-criteria

- K8s cluster should be in healthy state including desired worker nodes in ready state.
- CSPC and CVC operators, openebs-ndm, cstor-csi-node daemonsets and csi controler statefulsets should be in running state.
- storage class and the cstor volume policy with affinity labe should be created

## Exit-criteria

- Application should be deployed successfully. Application and the cstor target has to be scheduled on the same node. 

## Steps performed

- Create the cstor volume policy and storage class.
- Deploy the application using the storage cass that created.
- Verify the application pod cstor volume target pod created successfully.
- Check the applicatio pod and target pod scheduled on the same node. 
- Restart the application pod and target pod to verify it's scheduled on the same node.

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
kubectl logs -f <csi-app-target-affinity-xxxxx-xxxxx> -n e2e
```
To get the test-case result, get the corresponding e2e custom-resource `e2eresult` (short name: e2er ) and check its phase (Running or Completed) and result (Pass or Fail).

```
kubectl get e2er
kubectl get e2er csi-app-target-affinity -n e2e --no-headers -o custom-columns=:.spec.testStatus.phase
kubectl get e2er csi-app-target-affinity -n e2e --no-headers -o custom-columns=:.spec.testStatus.result
```