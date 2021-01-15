# CStor Helm Repository

<img width="300" align="right" alt="OpenEBS Logo" src="https://raw.githubusercontent.com/cncf/artwork/master/projects/openebs/stacked/color/openebs-stacked-color.png" xmlns="http://www.w3.org/1999/html">

[Helm3](https://helm.sh) must be installed to use the charts.
Please refer to Helm's [documentation](https://helm.sh/docs/) to get started.

Once Helm is set up properly, add the repo as follows:

```bash
$ helm repo add cstor https://openebs.github.io/cstor-operators
```

You can then run `helm search repo cstor` to see the charts.

#### Update OpenEBS CStor Repo

Once cstor repository has been successfully fetched into the local system, it has to be updated to get the latest version. The cstor repo can be updated using the following command.

```bash
helm repo update
```

#### Install using Helm 3

- Assign openebs namespace to the current context:
```bash
kubectl config set-context <current_context_name> --namespace=openebs
```

- If namespace is not created, run the following command
```bash
helm install <your-relase-name> cstor/openebs-cstor --create-namespace
```
- Else, if namespace is already created, run the following command
```bash
helm install <your-relase-name> cstor/openebs-cstor
```

#### Upgrade OpenEBS CStor

- Upgrade the CRDs by applying the CRD yaml from the helm repo 
```bash
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/cstorbackup.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/cstorcompletedbackup.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/cstorpoolcluster.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/cstorpoolinstance.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/cstorrestore.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/cstorvolume.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/cstorvolumeattachment.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/cstorvolumeconfig.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/cstorvolumepolicy.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/cstorvolumereplica.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/migrationtask.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/upgradetask.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/volumesnapshot.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/volumesnapshotclass.yaml
kubectl apply -f https://raw.githubusercontent.com/openebs/cstor-operators/master/deploy/helm/charts/crds/volumesnapshotcontent.yaml
```

- Upgrade the CStor control-plane
```bash
helm upgrade <your-release-name> cstor/openebs-cstor --namespace=openebs
```

- To upgrade the data plane components follow [this](https://github.com/openebs/upgrade/blob/master/docs/upgrade.md) doc. 