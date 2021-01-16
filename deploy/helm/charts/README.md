# OpenEBS CStor

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![Release Charts](https://github.com/openebs/cstor-operators/workflows/Release%20Charts/badge.svg?branch=master)
![Chart Lint and Test](https://github.com/openebs/cstor-operators/workflows/Chart%20Lint%20and%20Test/badge.svg)

OpenEBS CStor helm chart for Kubernetes. This chart bootstraps OpenEBS cstor operators and csi driver deployment on a [Kubernetes](http://kubernetes.io) cluster using the  [Helm](https://helm.sh) package manager

**Homepage:** <http://www.openebs.io/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| kiranmova | kiran.mova@mayadata.io |  |
| prateekpandey14 | prateek.pandey@mayadata.io |  |
| sonasingh46 | sonasingh46@gmail.com |  |

## Get Repo Info

```console
helm repo add openebs-cstor https://openebs.github.io/cstor-operators
helm repo update
```

_See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation._

## Install Chart

Please visit the [link](https://openebs.github.io/cstor-operators) for install instructions via helm3.

```console
# Helm
$ helm install [RELEASE_NAME] openebs-cstor/cstor
```

_See [configuration](#configuration) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._


## Dependencies

By default this chart installs additional, dependent charts:

| Repository | Name | Version |
|------------|------|---------|
| https://openebs.github.io/node-disk-manager | openebs-ndm | 1.1.1 |

To disable the dependency during installation, set `openebsNDM.enabled` to `false`.

_See [helm dependency](https://helm.sh/docs/helm/helm_dependency/) for command documentation._

## Uninstall Chart

```console
# Helm
$ helm uninstall [RELEASE_NAME]
```

This removes all the Kubernetes components associated with the chart and deletes the release.

_See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation._

## Upgrading Chart

```console
# Helm
$ helm upgrade [RELEASE_NAME] [CHART] --install
```

## Configuration

The following table lists the configurable parameters of the OpenEBS CStor chart and their default values.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| admissionServer.annotations | object | `{}` | Admission webhook annotations |
| admissionServer.componentName | string | `"cstor-admission-webhook"` | Admission webhook Component Name |
| admissionServer.failurePolicy | string | `"Fail"` | Admission Webhook failure policy |
| admissionServer.image.pullPolicy | string | `"IfNotPresent"` | Admission webhook image pull policy |
| admissionServer.image.registry | string | `nil` | Admission webhook image registry |
| admissionServer.image.repository | string | `"openebs/cstor-webhook"` | Admission webhook image repo |
| admissionServer.image.tag | string | `"2.5.0"` | Admission webhook image tag |
| admissionServer.nodeSelector | object | `{}` | Admission webhook pod node selector |
| admissionServer.podAnnotations | object | `{}` |  Admission webhook pod annotations |
| admissionServer.resources | object | `{}` | Admission webhook pod resources |
| admissionServer.securityContext | object | `{}` | Admission webhook security context |
| admissionServer.tolerations | list | `[]` | Admission webhook tolerations |
| csiController.annotations | object | `{}` | CSI controller annotations |
| csiController.attacher.image.pullPolicy | string | `"IfNotPresent"` | CSI attacher image pull policy |
| csiController.attacher.image.registry | string | `"quay.io/"` |  CSI attacher image registry |
| csiController.attacher.image.repository | string | `"k8scsi/csi-attacher"` |  CSI attacher image repo |
| csiController.attacher.image.tag | string | `"v3.1.0"` | CSI attacher image tag |
| csiController.attacher.name | string | `"csi-attacher"` |  CSI attacher container name|
| csiController.componentName | string | `""` | CSI controller component name |
| csiController.driverRegistrar.image.pullPolicy | string | `"IfNotPresent"` | CSI driver registrar image pull policy  |
| csiController.driverRegistrar.image.registry | string | `"quay.io/"` | CSI driver registrar image registry |
| csiController.driverRegistrar.image.repository | string | `"k8scsi/csi-cluster-driver-registrar"` | CSI driver registrar image repo |
| csiController.driverRegistrar.image.tag | string | `"v1.0.1"` |  CSI driver registrar image tag|
| csiController.driverRegistrar.name | string | `"csi-cluster-driver-registrar"` | CSI driver registrar container name  |
| csiController.nodeSelector | object | `{}` |  CSI controller pod node selector |
| csiController.podAnnotations | object | `{}` | CSI controller pod annotations |
| csiController.provisioner.image.pullPolicy | string | `"IfNotPresent"` | CSI provisioner image pull policy |
| csiController.provisioner.image.registry | string | `"quay.io/"` | CSI provisioner image pull registry |
| csiController.provisioner.image.repository | string | `"k8scsi/csi-provisioner"` | CSI provisioner image pull repository |
| csiController.provisioner.image.tag | string | `"v2.1.0"` | CSI provisioner image tag |
| csiController.provisioner.name | string | `"csi-provisioner"` | CSI provisioner container name |
| csiController.resizer.image.pullPolicy | string | `"IfNotPresent"` | CSI resizer image pull policy  |
| csiController.resizer.image.registry | string | `"quay.io/"` | CSI resizer image registry |
| csiController.resizer.image.repository | string | `"k8scsi/csi-resizer"` |  CSI resizer image repository|
| csiController.resizer.image.tag | string | `"v1.1.0"` | CSI resizer image tag |
| csiController.resizer.name | string | `"csi-resizer"` | CSI resizer container name |
| csiController.resources | object | `{}` | CSI controller container resources |
| csiController.securityContext | object | `{}` | CSI controller security context |
| csiController.snapshotController.image.pullPolicy | string | `"IfNotPresent"` | CSI snapshot controller image pull policy |
| csiController.snapshotController.image.registry | string | `"quay.io/"` | CSI snapshot controller image registry |
| csiController.snapshotController.image.repository | string | `"k8scsi/snapshot-controller"` | CSI snapshot controller image repository |
| csiController.snapshotController.image.tag | string | `"v3.0.3"` | CSI snapshot controller image tag |
| csiController.snapshotController.name | string | `"snapshot-controller"` | CSI snapshot controller container name |
| csiController.snapshotter.image.pullPolicy | string | `"IfNotPresent"` | CSI snapshotter image pull policy |
| csiController.snapshotter.image.registry | string | `"quay.io/"` | CSI snapshotter image pull registry |
| csiController.snapshotter.image.repository | string | `"k8scsi/csi-snapshotter"` | CSI snapshotter image repositroy |
| csiController.snapshotter.image.tag | string | `"v3.0.3"` | CSI snapshotter image tag |
| csiController.snapshotter.name | string | `"csi-snapshotter"` | CSI snapshotter container name |
| csiController.tolerations | list | `[]` | CSI controller pod tolerations |
| csiNode.annotations | object | `{}` | CSI Node annotations |
| csiNode.componentName | string | `"openebs-cstor-csi-node"` | CSI Node component name |
| csiNode.driverRegistrar.image.pullPolicy | string | `"IfNotPresent"` | CSI Node driver registrar image pull policy|
| csiNode.driverRegistrar.image.registry | string | `"quay.io/"` | CSI Node driver registrar image registry |
| csiNode.driverRegistrar.image.repository | string | `"k8scsi/csi-node-driver-registrar"` | CSI Node driver registrar image repository |
| csiNode.driverRegistrar.image.tag | string | `"v1.0.1"` |  CSI Node driver registrar image tag|
| csiNode.driverRegistrar.name | string | `"csi-node-driver-registrar"` | CSI Node driver registrar container name |
| csiNode.kubeletDir | string | `"/var/lib/kubelet/"` | Kubelet root dir |
| csiNode.labels | object | `{}` | CSI Node pod labels |
| csiNode.nodeSelector | object | `{}` |   CSI Node pod nodeSelector |
| csiNode.podAnnotations | object | `{}` | CSI Node pod annotations |
| csiNode.resources | object | `{}` | CSI Node pod resources |
| csiNode.securityContext | object | `{}` | CSI Node pod security context |
| csiNode.tolerations | list | `[]` | CSI Node pod tolerations |
| csiNode.updateStrategy.type | string | `"RollingUpdate"` | CSI Node daemonset update strategy |
| cspcOperator.annotations | object | `{}` | CSPC operator annotations |
| cspcOperator.componentName | string | `"cspc-operator"` | CSPC operator component name |
| cspcOperator.cstorPool.image.registry | string | `nil` | CStor pool image registry |
| cspcOperator.cstorPool.image.repository | string | `"openebs/cstor-pool"` | CStor pool image repository|
| cspcOperator.cstorPool.image.tag | string | `"2.5.0"` | CStor pool image tag |
| cspcOperator.cstorPoolExporter.image.registry | string | `nil` | CStor pool exporter image registry |
| cspcOperator.cstorPoolExporter.image.repository | string | `"openebs/m-exporter"` | CStor pool exporter image repositry |
| cspcOperator.cstorPoolExporter.image.tag | string | `"2.5.0"` | CStor pool exporter image tag |
| cspcOperator.image.pullPolicy | string | `"IfNotPresent"` | CSPC operator image pull policy |
| cspcOperator.image.registry | string | `nil` | CSPC operator image registry |
| cspcOperator.image.repository | string | `"openebs/cspc-operator"` | CSPC operator image repository |
| cspcOperator.image.tag | string | `"2.5.0"` |  CSPC operator image tag |
| cspcOperator.nodeSelector | object | `{}` |  CSPC operator pod nodeSelector|
| cspcOperator.podAnnotations | object | `{}` | CSPC operator pod annotations |
| cspcOperator.poolManager.image.registry | string | `nil` | CStor Pool Manager image registry  |
| cspcOperator.poolManager.image.repository | string | `"openebs/cstor-pool-manager"` | CStor Pool Manager image repository |
| cspcOperator.poolManager.image.tag | string | `"2.5.0"` | CStor Pool Manager image tag |
| cspcOperator.resources | object | `{}` | CSPC operator pod resources |
| cspcOperator.resyncInterval | string | `"30"` | CSPC operator resync interval |
| cspcOperator.securityContext | object | `{}` | CSPC operator security context |
| cspcOperator.tolerations | list | `[]` | CSPC operator pod tolerations |
| cstorCSIPlugin.image.pullPolicy | string | `"IfNotPresent"` | CStor CSI driver image pull policy |
| cstorCSIPlugin.image.registry | string | `nil` | CStor CSI driver image registry |
| cstorCSIPlugin.image.repository | string | `"openebs/cstor-csi-driver"` |  CStor CSI driver image repository |
| cstorCSIPlugin.image.tag | string | `"2.5.0"` | CStor CSI driver image tag |
| cstorCSIPlugin.name | string | `"cstor-csi-plugin"` | CStor CSI driver container name |
| cvcOperator.annotations | object | `{}` | CVC operator annotations |
| cvcOperator.componentName | string | `"cvc-operator"` | CVC operator component name |
| cvcOperator.image.pullPolicy | string | `"IfNotPresent"` | CVC operator image pull policy  |
| cvcOperator.image.registry | string | `nil` | CVC operator image registry |
| cvcOperator.image.repository | string | `"openebs/cvc-operator"` | CVC operator image repository |
| cvcOperator.image.tag | string | `"2.5.0"` | CVC operator image tag |
| cvcOperator.nodeSelector | object | `{}` | CVC operator pod nodeSelector |
| cvcOperator.podAnnotations | object | `{}` | CVC operator pod annotations |
| cvcOperator.resources | object | `{}` |CVC operator pod resources  |
| cvcOperator.resyncInterval | string | `"30"` | CVC operator resync interval |
| cvcOperator.securityContext | object | `{}` | CVC operator security context |
| cvcOperator.target.image.registry | string | `nil` | Volume Target image registry  |
| cvcOperator.target.image.repository | string | `"openebs/cstor-istgt"` | Volume Target image repository |
| cvcOperator.target.image.tag | string | `"2.5.0"` | Volume Target image tag |
| cvcOperator.tolerations | list | `[]` | CVC operator pod tolerations |
| cvcOperator.volumeExporter.image.registry | string | `nil` | Volume exporter image registry |
| cvcOperator.volumeExporter.image.repository | string | `"openebs/m-exporter"` | Volume exporter image repository |
| cvcOperator.volumeExporter.image.tag | string | `"2.5.0"` | Volume exporter image tag |
| cvcOperator.volumeMgmt.image.registry | string | `nil` | Volume mgmt image registry |
| cvcOperator.volumeMgmt.image.repository | string | `"openebs/cstor-volume-manager"` | Volume mgmt image repository |
| cvcOperator.volumeMgmt.image.tag | string | `"2.5.0"` |  Volume mgmt image tag|
| imagePullSecrets | string | `nil` | Image registry pull secrets |
| openebsNDM.enabled | bool | `true` | Enable OpenEBS NDM dependency |
| rbac.create | bool | `true` | Enable RBAC |
| rbac.pspEnabled | bool | `false` | Enable PodSecurityPolicy |
| release.version | string | `"2.5.0"` | Openebs CStor release version |
| serviceAccount.annotations | object | `{}` | Service Account annotations |
| serviceAccount.csiController.create | bool | `true` | Enable CSI Controller ServiceAccount |
| serviceAccount.csiController.name | string | `"openebs-cstor-csi-controller-sa"` | CSI Controller ServiceAccount name |
| serviceAccount.csiNode.create | bool | `true` | Enable CSI Node ServiceAccount |
| serviceAccount.csiNode.name | string | `"openebs-cstor-csi-node-sa"` | CSI Node ServiceAccount name |

