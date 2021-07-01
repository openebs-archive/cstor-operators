## About this experiment

This experiment deploys the cstor-operator provisioner in openebs namespace which includes csi controller cspc and cvc operator, cstor-csi-node,openebs-ndm deamonset.

## Entry-Criteria

- K8s cluster should be in healthy state including all the nodes in ready state.
- If we don't want to use this experiment to deploy cstor operator, we can directly apply the cstor operator file as mentioned below.

```
kubectl apply -f https://raw.githubusercontent.com/openebs/charts/gh-pages/<release-version>/cstor-operator.yaml
```

## Exit-Criteria

- cstor csi driver components should be deployed successfully and all the pods including csi-controller, cspc and cvc operator, openebs-ndm and csi node-agent daemonset are in running state.

## How to run

- This experiment accepts the parameters in form of kubernetes job environmental variables.
- For running this experiment of deploying cstor operator, clone openens/cstor-operators[https://github.com/openebs/cstor-operators] repo and then first apply rbac and crds for e2e-framework.
```
kubectl apply -f cstor-operators/e2e-tests/hack/rbac.yaml
kubectl apply -f cstor-operators/e2e-tests/hack/crds.yaml
```
then update the needed test specific values in run_e2e_test.yml file and create the kubernetes job.
```
kubectl create -f run_e2e_test.yml
```
All the env variables description is provided with the comments in the same file.

