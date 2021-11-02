/*
Copyright 2020 The OpenEBS Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package algorithm

import (
	"os"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	deployapi "github.com/openebs/api/v3/pkg/kubernetes/apps"
	coreapi "github.com/openebs/api/v3/pkg/kubernetes/core"
	util "github.com/openebs/api/v3/pkg/util"
	"github.com/openebs/cstor-operators/pkg/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PoolMgmtContainerName is the name of cstor target container name
	PoolMgmtContainerName = "cstor-pool-mgmt"

	// PoolContainerName is the name of cstor target container name
	PoolContainerName = "cstor-pool"

	// PoolExporterContainerName is the name of cstor target container name
	PoolExporterContainerName = "maya-exporter"
)

var (
	// run container in privileged mode configuration that will be
	// applied to a container.
	privileged            = true
	defaultPoolMgmtMounts = []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      "device",
			MountPath: "/dev",
		},
		corev1.VolumeMount{
			Name:      "tmp",
			MountPath: "/tmp",
		},
		corev1.VolumeMount{
			Name:      "udev",
			MountPath: "/run/udev",
		},
		corev1.VolumeMount{
			Name:      "storagepath",
			MountPath: "/var/openebs/cstor-pool",
		},
		corev1.VolumeMount{
			Name:      "sockfile",
			MountPath: "/var/tmp/sock",
		},
	}
	// hostpathType represents the hostpath type
	hostpathTypeDirectory = corev1.HostPathDirectory

	// hostpathType represents the hostpath type
	hostpathTypeDirectoryOrCreate = corev1.HostPathDirectoryOrCreate
)

// GetPoolDeploySpec returns the pool deployment spec.
func (c *Config) GetPoolDeploySpec(cspi *cstor.CStorPoolInstance) *appsv1.Deployment {
	deployObj := deployapi.NewDeployment().
		WithName(cspi.Name).
		WithNamespace(cspi.Namespace).
		WithAnnotationsNew(getDeployAnnotations()).
		WithLabelsNew(getDeployLabels(cspi)).
		WithOwnerReferenceNew(getDeployOwnerReference(cspi)).
		WithReplicas(getReplicaCount()).
		WithStrategyType(appsv1.RecreateDeploymentStrategyType).
		WithSelectorMatchLabelsNew(getDeployMatchLabels()).
		WithPodTemplateSpec(
			coreapi.NewPodTemplateSpec().
				WithLabelsNew(getPodLabels(cspi)).
				WithPriorityClassName(getPriorityClass(cspi)).
				WithNodeSelector(cspi.Spec.NodeSelector).
				WithAnnotationsNew(getPodAnnotations()).
				WithServiceAccountName(util.GetServiceAccountName()).
				WithTolerations(getPoolPodToleration(cspi)...).
				WithImagePullSecrets(coreapi.GetImagePullSecrets(util.GetOpenEBSImagePullSecrets())).
				WithContainers(
					coreapi.NewContainer().
						WithImage(getPoolMgmtImage()).
						WithName(PoolMgmtContainerName).
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithPrivilegedSecurityContext(&privileged).
						WithEnvsNew(getPoolMgmtEnv(cspi)).
						WithEnvs(getPoolUIDAsEnv(c.CSPC)).
						WithResources(getAuxResourceRequirement(cspi)).
						WithVolumeMountsNew(getPoolMgmtMounts()),
					coreapi.NewContainer().
						WithImage(getPoolImage()).
						WithName(PoolContainerName).
						WithResources(getResourceRequirementForCStorPool(cspi)).
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithPrivilegedSecurityContext(&privileged).
						WithPortsNew(getContainerPort(12000, 3232, 3233)).
						WithLivenessProbe(getPoolLivenessProbe()).
						WithEnvsNew(getPoolEnv(cspi)).
						WithEnvs(getPoolUIDAsEnv(c.CSPC)).
						WithLifeCycle(getPoolLifeCycle()).
						WithVolumeMountsNew(getPoolMounts()),
					coreapi.NewContainer().
						WithImage(getMayaExporterImage()).
						WithName(PoolExporterContainerName).
						// TODO: add default values for resources
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithPrivilegedSecurityContext(&privileged).
						WithPortsNew(getContainerPort(9500)).
						WithCommandNew([]string{"maya-exporter"}).
						WithArgumentsNew([]string{"-e=pool"}).
						WithVolumeMountsNew(getPoolMounts()),
				).
				WithVolumes(
					coreapi.NewVolume().
						WithName("device").
						WithHostPathAndType(
							"/dev",
							&hostpathTypeDirectory,
						),
					coreapi.NewVolume().
						WithName("udev").
						WithHostPathAndType(
							"/run/udev",
							&hostpathTypeDirectory,
						),
					coreapi.NewVolume().
						WithName("tmp").
						WithHostPathAndType(
							// NS + CstorPool + Name
							// ToDo: Add Namespace
							util.GetOpenebsBaseDirPath()+"/cstor-pool/"+c.CSPC.Name,
							&hostpathTypeDirectoryOrCreate,
						),
					coreapi.NewVolume().
						WithName("sparse").
						WithHostPathAndType(
							getSparseDirPath(),
							&hostpathTypeDirectoryOrCreate,
						),
					coreapi.NewVolume().
						WithName("storagepath").
						WithHostPathAndType(
							// NS + CstorPool + Name
							// ToDo: Add Namespace
							util.GetOpenebsBaseDirPath()+"/cstor-pool/"+c.CSPC.Name,
							&hostpathTypeDirectoryOrCreate,
						),
					coreapi.NewVolume().
						WithName("sockfile").
						WithEmptyDir(&corev1.EmptyDirVolumeSource{}),
				),
		).Build()

	return deployObj

}

func getReplicaCount() *int32 {
	var count int32 = 1
	return &count
}

func getDeployOwnerReference(cspi *cstor.CStorPoolInstance) []metav1.OwnerReference {
	OwnerReference := []metav1.OwnerReference{
		*metav1.NewControllerRef(cspi, cstor.SchemeGroupVersion.WithKind("CStorPoolInstance")),
	}
	return OwnerReference
}

// TODO: Use builder for labels and annotations
func getDeployLabels(csp *cstor.CStorPoolInstance) map[string]string {
	return map[string]string{
		string(types.CStorPoolClusterLabelKey):  csp.Labels[types.CStorPoolClusterLabelKey],
		"app":                                   "cstor-pool",
		string(types.CStorPoolInstanceLabelKey): csp.Name,
		"openebs.io/version":                    version.GetVersion(),
	}
}

func getDeployAnnotations() map[string]string {
	return map[string]string{
		"openebs.io/monitoring": "pool_exporter_prometheus",
	}
}

func getPodLabels(csp *cstor.CStorPoolInstance) map[string]string {
	return getDeployLabels(csp)
}

func getPodAnnotations() map[string]string {
	return map[string]string{
		"openebs.io/monitoring":                          "pool_exporter_prometheus",
		"prometheus.io/path":                             "/metrics",
		"prometheus.io/port":                             "9500",
		"prometheus.io/scrape":                           "true",
		"cluster-autoscaler.kubernetes.io/safe-to-evict": "false",
	}
}

func getDeployMatchLabels() map[string]string {
	return map[string]string{
		"app": "cstor-pool",
	}
}

// getVolumeTargetImage returns Volume target image
// retrieves the value of the environment variable named
// by the key.
func getPoolMgmtImage() string {
	image, present := os.LookupEnv("OPENEBS_IO_CSPI_MGMT_IMAGE")
	if !present {
		image = "quay.io/openebs/cstor-pool-manager-amd64:ci"
	}
	return image
}

// getVolumeTargetImage returns Volume target image
// retrieves the value of the environment variable named
// by the key.
func getPoolImage() string {
	image, present := os.LookupEnv("OPENEBS_IO_CSTOR_POOL_IMAGE")
	if !present {
		image = "quay.io/openebs/cstor-pool:ci"
	}
	return image
}

// getVolumeTargetImage returns Volume target image
// retrieves the value of the environment variable named
// by the key.
func getMayaExporterImage() string {
	image, present := os.LookupEnv("OPENEBS_IO_CSTOR_POOL_EXPORTER_IMAGE")
	if !present {
		image = "quay.io/openebs/m-exporter:ci"
	}
	return image
}

func getContainerPort(port ...int32) []corev1.ContainerPort {
	var containerPorts []corev1.ContainerPort
	for _, p := range port {
		containerPorts = append(containerPorts, corev1.ContainerPort{ContainerPort: p, Protocol: "TCP"})
	}
	return containerPorts
}

func getPoolMgmtMounts() []corev1.VolumeMount {
	return append(
		defaultPoolMgmtMounts,
		corev1.VolumeMount{
			Name:      "sparse",
			MountPath: getSparseDirPath(),
		},
	)
}

func getSparseDirPath() string {
	dir, present := os.LookupEnv("OPENEBS_IO_CSTOR_POOL_SPARSE_DIR")
	if !present {
		dir = "/var/openebs/sparse"
	}
	return dir
}

func getPoolUIDAsEnv(cspc *cstor.CStorPoolCluster) []corev1.EnvVar {
	var env []corev1.EnvVar
	return append(
		env,
		corev1.EnvVar{
			Name:  "OPENEBS_IO_POOL_NAME",
			Value: string(cspc.GetUID()),
		},
	)
}

func getPoolMgmtEnv(cspi *cstor.CStorPoolInstance) []corev1.EnvVar {
	var env []corev1.EnvVar
	return append(
		env,
		corev1.EnvVar{
			Name:  "OPENEBS_IO_CSPI_ID",
			Value: string(cspi.GetUID()),
		},
		corev1.EnvVar{
			Name: "RESYNC_INTERVAL",
			// TODO : Add tunable
			Value: "30",
		},
		corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		corev1.EnvVar{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	)
}

func getPoolLivenessProbe() *corev1.Probe {
	probe := &corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/sh", "-c", "timeout 120 zfs set io.openebs:livenesstimestamp=\"$(date +%s)\" cstor-$OPENEBS_IO_POOL_NAME"},
			},
		},
		FailureThreshold:    3,
		InitialDelaySeconds: 300,
		PeriodSeconds:       60,
		TimeoutSeconds:      150,
	}
	return probe
}

func getPoolMounts() []corev1.VolumeMount {
	return getPoolMgmtMounts()
}

func getPoolEnv(cspi *cstor.CStorPoolInstance) []corev1.EnvVar {
	var env []corev1.EnvVar
	return append(
		env,
		corev1.EnvVar{
			Name:  "OPENEBS_IO_CSTOR_ID",
			Value: string(cspi.GetUID()),
		},
	)
}

func getPoolLifeCycle() *corev1.Lifecycle {
	lc := &corev1.Lifecycle{
		PostStart: &corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/sh", "-c", "sleep 2"},
			},
		},
	}
	return lc
}

// getResourceRequirementForCStorPool returns resource requirement for cstor pool container.
func getResourceRequirementForCStorPool(cspi *cstor.CStorPoolInstance) corev1.ResourceRequirements {
	var resourceRequirements corev1.ResourceRequirements
	if cspi.Spec.PoolConfig.Resources == nil {
		resourceRequirements = corev1.ResourceRequirements{}
	} else {
		resourceRequirements = *cspi.Spec.PoolConfig.Resources
	}
	return resourceRequirements
}

// getAuxResourceRequirement returns resource requirement for cstor pool side car containers.
func getAuxResourceRequirement(cspi *cstor.CStorPoolInstance) corev1.ResourceRequirements {
	var auxResourceRequirements corev1.ResourceRequirements
	if cspi.Spec.PoolConfig.AuxResources == nil {
		auxResourceRequirements = corev1.ResourceRequirements{}
	} else {
		auxResourceRequirements = *cspi.Spec.PoolConfig.AuxResources
	}
	return auxResourceRequirements
}

// getPoolPodToleration returns pool pod tolerations.
func getPoolPodToleration(cspi *cstor.CStorPoolInstance) []corev1.Toleration {
	var tolerations []corev1.Toleration
	if len(cspi.Spec.PoolConfig.Tolerations) == 0 {
		tolerations = []corev1.Toleration{}
	} else {
		tolerations = cspi.Spec.PoolConfig.Tolerations
	}
	return tolerations
}

func getPriorityClass(cspi *cstor.CStorPoolInstance) string {
	if cspi.Spec.PoolConfig.PriorityClassName == nil {
		return ""
	}
	return *cspi.Spec.PoolConfig.PriorityClassName
}
