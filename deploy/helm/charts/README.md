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
| https://openebs.github.io/node-disk-manager | openebs-ndm | 1.2.0 |

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
| admissionServer.image.tag | string | `"2.6.0"` | Admission webhook image tag |
| admissionServer.nodeSelector | object | `{}` | Admission webhook pod node selector |
| admissionServer.podAnnotations | object | `{}` |  Admission webhook pod annotations |
| admissionServer.resources | object | `{}` | Admission webhook pod resources |
| admissionServer.securityContext | object | `{}` | Admission webhook security context |
| admissionServer.tolerations | list | `[]` | Admission webhook tolerations |
| csiController.annotations | object | `{}` | CSI controller annotations |
| csiController.attacher.image.pullPolicy | string | `"IfNotPresent"` | CSI attacher image pull policy |
| csiController.attacher.image.registry | string | `"k8s.gcr.io/"` |  CSI attacher image registry |
| csiController.attacher.image.repository | string | `"sig-storage/csi-attacher"` |  CSI attacher image repo |
| csiController.attacher.image.tag | string | `"v3.1.0"` | CSI attacher image tag |
| csiController.attacher.name | string | `"csi-attacher"` |  CSI attacher container name|
| csiController.componentName | string | `"openebs-cstor-csi-controller"` | CSI controller component name |
| csiController.nodeSelector | object | `{}` |  CSI controller pod node selector |
| csiController.podAnnotations | object | `{}` | CSI controller pod annotations |
| csiController.provisioner.image.pullPolicy | string | `"IfNotPresent"` | CSI provisioner image pull policy |
| csiController.provisioner.image.registry | string | `"k8s.gcr.io/"` | CSI provisioner image pull registry |
| csiController.provisioner.image.repository | string | `"sig-storage/csi-provisioner"` | CSI provisioner image pull repository |
| csiController.provisioner.image.tag | string | `"v2.1.0"` | CSI provisioner image tag |
| csiController.provisioner.name | string | `"csi-provisioner"` | CSI provisioner container name |
| csiController.resizer.image.pullPolicy | string | `"IfNotPresent"` | CSI resizer image pull policy  |
| csiController.resizer.image.registry | string | `"k8s.gcr.io/"` | CSI resizer image registry |
| csiController.resizer.image.repository | string | `"sig-storage/csi-resizer"` |  CSI resizer image repository|
| csiController.resizer.image.tag | string | `"v1.1.0"` | CSI resizer image tag |
| csiController.resizer.name | string | `"csi-resizer"` | CSI resizer container name |
| csiController.resources | object | `{}` | CSI controller container resources |
| csiController.securityContext | object | `{}` | CSI controller security context |
| csiController.snapshotController.image.pullPolicy | string | `"IfNotPresent"` | CSI snapshot controller image pull policy |
| csiController.snapshotController.image.registry | string | `"k8s.gcr.io/"` | CSI snapshot controller image registry |
| csiController.snapshotController.image.repository | string | `"sig-storage/snapshot-controller"` | CSI snapshot controller image repository |
| csiController.snapshotController.image.tag | string | `"v3.0.3"` | CSI snapshot controller image tag |
| csiController.snapshotController.name | string | `"snapshot-controller"` | CSI snapshot controller container name |
| csiController.snapshotter.image.pullPolicy | string | `"IfNotPresent"` | CSI snapshotter image pull policy |
| csiController.snapshotter.image.registry | string | `"k8s.gcr.io/"` | CSI snapshotter image pull registry |
| csiController.snapshotter.image.repository | string | `"sig-storage/csi-snapshotter"` | CSI snapshotter image repositroy |
| csiController.snapshotter.image.tag | string | `"v3.0.3"` | CSI snapshotter image tag |
| csiController.snapshotter.name | string | `"csi-snapshotter"` | CSI snapshotter container name |
| csiController.tolerations | list | `[]` | CSI controller pod tolerations |
| csiNode.annotations | object | `{}` | CSI Node annotations |
| csiNode.componentName | string | `"openebs-cstor-csi-node"` | CSI Node component name |
| csiNode.driverRegistrar.image.pullPolicy | string | `"IfNotPresent"` | CSI Node driver registrar image pull policy|
| csiNode.driverRegistrar.image.registry | string | `"k8s.gcr.io/"` | CSI Node driver registrar image registry |
| csiNode.driverRegistrar.image.repository | string | `"sig-storage/csi-node-driver-registrar"` | CSI Node driver registrar image repository |
| csiNode.driverRegistrar.image.tag | string | `"v2.1.0"` |  CSI Node driver registrar image tag|
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
| cspcOperator.cstorPool.image.tag | string | `"2.6.0"` | CStor pool image tag |
| cspcOperator.cstorPoolExporter.image.registry | string | `nil` | CStor pool exporter image registry |
| cspcOperator.cstorPoolExporter.image.repository | string | `"openebs/m-exporter"` | CStor pool exporter image repositry |
| cspcOperator.cstorPoolExporter.image.tag | string | `"2.6.0"` | CStor pool exporter image tag |
| cspcOperator.image.pullPolicy | string | `"IfNotPresent"` | CSPC operator image pull policy |
| cspcOperator.image.registry | string | `nil` | CSPC operator image registry |
| cspcOperator.image.repository | string | `"openebs/cspc-operator"` | CSPC operator image repository |
| cspcOperator.image.tag | string | `"2.6.0"` |  CSPC operator image tag |
| cspcOperator.nodeSelector | object | `{}` |  CSPC operator pod nodeSelector|
| cspcOperator.podAnnotations | object | `{}` | CSPC operator pod annotations |
| cspcOperator.poolManager.image.registry | string | `nil` | CStor Pool Manager image registry  |
| cspcOperator.poolManager.image.repository | string | `"openebs/cstor-pool-manager"` | CStor Pool Manager image repository |
| cspcOperator.poolManager.image.tag | string | `"2.6.0"` | CStor Pool Manager image tag |
| cspcOperator.resources | object | `{}` | CSPC operator pod resources |
| cspcOperator.resyncInterval | string | `"30"` | CSPC operator resync interval |
| cspcOperator.securityContext | object | `{}` | CSPC operator security context |
| cspcOperator.tolerations | list | `[]` | CSPC operator pod tolerations |
| cstorCSIPlugin.image.pullPolicy | string | `"IfNotPresent"` | CStor CSI driver image pull policy |
| cstorCSIPlugin.image.registry | string | `nil` | CStor CSI driver image registry |
| cstorCSIPlugin.image.repository | string | `"openebs/cstor-csi-driver"` |  CStor CSI driver image repository |
| cstorCSIPlugin.image.tag | string | `"2.6.0"` | CStor CSI driver image tag |
| cstorCSIPlugin.name | string | `"cstor-csi-plugin"` | CStor CSI driver container name |
| cstorCSIPlugin.remount | string | `"true"` | Enable/disable auto-remount when volume recovers from read-only state |
| cvcOperator.annotations | object | `{}` | CVC operator annotations |
| cvcOperator.componentName | string | `"cvc-operator"` | CVC operator component name |
| cvcOperator.image.pullPolicy | string | `"IfNotPresent"` | CVC operator image pull policy  |
| cvcOperator.image.registry | string | `nil` | CVC operator image registry |
| cvcOperator.image.repository | string | `"openebs/cvc-operator"` | CVC operator image repository |
| cvcOperator.image.tag | string | `"2.6.0"` | CVC operator image tag |
| cvcOperator.nodeSelector | object | `{}` | CVC operator pod nodeSelector |
| cvcOperator.podAnnotations | object | `{}` | CVC operator pod annotations |
| cvcOperator.resources | object | `{}` |CVC operator pod resources  |
| cvcOperator.resyncInterval | string | `"30"` | CVC operator resync interval |
| cvcOperator.securityContext | object | `{}` | CVC operator security context |
| cvcOperator.target.image.registry | string | `nil` | Volume Target image registry  |
| cvcOperator.target.image.repository | string | `"openebs/cstor-istgt"` | Volume Target image repository |
| cvcOperator.target.image.tag | string | `"2.6.0"` | Volume Target image tag |
| cvcOperator.tolerations | list | `[]` | CVC operator pod tolerations |
| cvcOperator.volumeExporter.image.registry | string | `nil` | Volume exporter image registry |
| cvcOperator.volumeExporter.image.repository | string | `"openebs/m-exporter"` | Volume exporter image repository |
| cvcOperator.volumeExporter.image.tag | string | `"2.6.0"` | Volume exporter image tag |
| cvcOperator.volumeMgmt.image.registry | string | `nil` | Volume mgmt image registry |
| cvcOperator.volumeMgmt.image.repository | string | `"openebs/cstor-volume-manager"` | Volume mgmt image repository |
| cvcOperator.volumeMgmt.image.tag | string | `"2.6.0"` |  Volume mgmt image tag|
| imagePullSecrets | string | `nil` | Image registry pull secrets |
| openebsNDM.enabled | bool | `true` | Enable OpenEBS NDM dependency |
| openebs-ndm.featureGates.APIService.enabled | bool | `true` | Enable 'API Service' feature gate for NDM |
| openebs-ndm.featureGates.APIService.featureGateFlag | string | `"APIService"` | 'API Service' feature gate flag for NDM  |
| openebs-ndm.featureGates.APIService.address | string | `true` | 'API Service' feature gate address for NDM |
| openebs-ndm.featureGates.enabled | bool | `true` | Enable NDM feature gates  |
| openebs-ndm.featureGates.GPTBasedUUID.enabled | bool | `true` | Enable 'GPT-based UUID' feature gate for NDM |
| openebs-ndm.featureGates.GPTBasedUUID.featureGateFlag | string | `"GPTBasedUUID"` | 'GPT-based UUID' feature gate flag for NDM |
| openebs-ndm.featureGates.UseOSDisk.enabled | bool | `true` | Enable 'Use OS-disk' feature gate for NDM |
| openebs-ndm.featureGates.UseOSDisk.featureGateFlag | string | `"UseOSDisk"` | 'Use OS-disk' feature gate flag for NDM |
| openebs-ndm.helperPod.image.registry | string | `nil` | Registry for helper image |
| openebs-ndm.helperPod.image.repository | string | `openebs/linux-utils` | Image for helper pod |
| openebs-ndm.helperPod.image.pullPolicy | string | `"IfNotPresent"` | Pull policy for helper pod |
| openebs-ndm.helperPod.image.tag | string | `2.6.0` | Image tag for helper image |
| openebs-ndm.ndm.annotations | object | `{}` | Annotations for NDM daemonset metadata |
| openebs-ndm.ndm.componentName | string | `ndm` | Node Disk Manager component name |
| openebs-ndm.ndm.enabled | bool | `true` | Enable Node Disk Manager |
| openebs-ndm.ndm.filters.enableOsDiskExcludeFilter | bool | `true` | Enable filters of OS disk exclude |
| openebs-ndm.ndm.filters.enableVendorFilter | bool | `true` | Enable filters of venders |
| openebs-ndm.ndm.filters.excludeVendors | string | `"CLOUDBYT,OpenEBS"` | Exclude devices with specified vendor |
| openebs-ndm.ndm.filters.enablePathFilter | bool | `true` | Enable filters of paths |
| openebs-ndm.ndm.filters.includePaths | string | `""` | Include devices with specified path patterns |
| openebs-ndm.ndm.filters.excludePaths | string | `"loop,fd0,sr0,/dev/ram,/dev/dm-,/dev/md,/dev/rbd,/dev/zd"` | Exclude devices with specified path patterns |
| openebs-ndm.ndm.healthCheck.initialDelaySeconds | string | `30` | Delay before liveness probe is initiated |
| openebs-ndm.ndm.healthCheck.periodSeconds | string | `60` | How often to perform the liveness probe |
| openebs-ndm.ndm.image.registry | string | `nil` | Registry for Node Disk Manager image |
| openebs-ndm.ndm.image.repository | string | `openebs/node-disk-manager` | Image repository for Node Disk Manager |
| openebs-ndm.ndm.image.pullPolicy | string | `"IfNotPresent"` | Image pull policy for Node Disk Manager |
| openebs-ndm.ndm.image.tag | string | `1.2.0` | Image tag for Node Disk Manager |
| openebs-ndm.ndm.nodeSelector | object | `{}` | Nodeselector for daemonset pods |
| openebs-ndm.ndm.podAnnotations | object | `{}` | Annotations for NDM daemonset's pods metadata |
| openebs-ndm.ndm.podLabels | object | `{}` | Appends labels to the pods |
| openebs-ndm.ndm.probes.enableSeachest | bool | `false` | Enable Seachest probe for NDM |
| openebs-ndm.ndm.probes.enableUdevProbe | bool | `true` | Enable Udev probe for NDM |
| openebs-ndm.ndm.probes.enableSmartProbe | bool | `true` | Enable Smart probe for NDM |
| openebs-ndm.ndm.resources | object | `{}` | Resource and request and limit for containers |
| openebs-ndm.ndm.securityContext | object | `{}` | Seurity context for NDM daemonset container |
| openebs-ndm.ndm.sparse.count | string | `"0"` | Number of sparse files to be created |
| openebs-ndm.ndm.sparse.path | string | `"/var/openebs/sparse"` | Directory where Sparse files are created |
| openebs-ndm.ndm.sparse.size | string | `"10737418240"` | Size of the sparse file in bytes |
| openebs-ndm.ndm.tolerations | list | `[]` | NDM daemonset's pod toleration values |
| openebs-ndm.ndm.updateStrategy.type | string | `RollingUpdate` | Update strategy for NDM daemonset |
| openebs-ndm.ndmOperator.annotations | object | `{}` | Annotations for NDM operator metadata |
| openebs-ndm.ndmOperator.enabled | bool | `true` | Enable NDM Operator |
| openebs-ndm.ndmOperator.healthCheck.initialDelaySeconds | string | `30` | Delay before liveness probe is initiated |
| openebs-ndm.ndmOperator.healthCheck.periodSeconds | string | `60` | How often to perform the liveness probe |
| openebs-ndm.ndmOperator.image.registry | string | `nil` | Registry for NDM operator image |
| openebs-ndm.ndmOperator.image.repository | string | `openebs/node-disk-operator` | Image repository for NDM operator |
| openebs-ndm.ndmOperator.image.pullPolicy | string | `IfNotPresent` | Image pull policy for NDM operator |
| openebs-ndm.ndmOperator.image.tag | string | `1.2.0` |  Image tag for NDM operator |
| openebs-ndm.ndmOperator.nodeSelector | object | `{}` | Nodeselector for operator pods |
| openebs-ndm.ndmOperator.podAnnotations | object | `{}` | Annotations for NDM operator's pods metadata |
| openebs-ndm.ndmOperator.podLabels | object | `{}` | Appends labels to the pods |
| openebs-ndm.ndmOperator.readinessCheck.initialDelaySeconds | string | `4` | Delay before readiness probe is initiated |
| openebs-ndm.ndmOperator.readinessCheck.periodSeconds | string | `10` | How often to perform the readiness probe |
| openebs-ndm.ndmOperator.readinessCheck.failureThreshold | string | `1` | Failure threshold for the readiness probe |
| openebs-ndm.ndmOperator.replicas | string | `1` | Pod replica count for NDM operator |
| openebs-ndm.ndmOperator.resources | object | `{}` | Resource and request and limit for containers |
| openebs-ndm.ndmOperator.securityContext | object | `{}` | Seurity context for container |
| openebs-ndm.ndmOperator.tolerations | list | `[]` | NDM operator's pod toleration values |
| openebs-ndm.serviceAccount.create | bool | `true` | Create a service account or not |
| openebs-ndm.serviceAccount.name | string | `openebs-ndm` | Name for the service account |
| openebs-ndm.varDirectoryPath.baseDir | string | `"/var/openebs"` | Directory to store debug info and so forth |
| rbac.create | bool | `true` | Enable RBAC |
| rbac.pspEnabled | bool | `false` | Enable PodSecurityPolicy |
| release.version | string | `"2.6.0"` | Openebs CStor release version |
| serviceAccount.annotations | object | `{}` | Service Account annotations |
| serviceAccount.csiController.create | bool | `true` | Enable CSI Controller ServiceAccount |
| serviceAccount.csiController.name | string | `"openebs-cstor-csi-controller-sa"` | CSI Controller ServiceAccount name |
| serviceAccount.csiNode.create | bool | `true` | Enable CSI Node ServiceAccount |
| serviceAccount.csiNode.name | string | `"openebs-cstor-csi-node-sa"` | CSI Node ServiceAccount name |

