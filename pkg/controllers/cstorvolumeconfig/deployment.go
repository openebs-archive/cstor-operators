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

package cstorvolumeconfig

import (
	"context"
	"os"
	"strconv"

	apis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	deploy "github.com/openebs/api/v3/pkg/kubernetes/apps"
	apicore "github.com/openebs/api/v3/pkg/kubernetes/core"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/openebs/cstor-operators/pkg/version"
	errors "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (

	// tolerationSeconds represents the period of time the toleration
	// tolerates the taint
	tolerationSeconds = int64(30)
	// deployreplicas is replica count for target deployment
	deployreplicas int32 = 1

	// run container in privileged mode configuration that will be
	// applied to a container.
	privileged = true

	resyncInterval = "30"

	// MountPropagationBidirectional means that the volume in a container will
	// receive new mounts from the host or other containers, and its own mounts
	// will be propagated from the container to the host or other containers.
	// mountPropagation = corev1.MountPropagationBidirectional

	// hostpathType represents the hostpath type
	hostpathType = corev1.HostPathDirectoryOrCreate

	defaultMounts = []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      "sockfile",
			MountPath: "/var/run",
		},
		corev1.VolumeMount{
			Name:      "conf",
			MountPath: "/usr/local/etc/istgt",
		},
		corev1.VolumeMount{
			Name:      "storagepath",
			MountPath: "/var/openebs/cstor-target",
		},
	}
	// TargetContainerName is the name of cstor target container name
	TargetContainerName = "cstor-istgt"
	// MonitorContainerName is the name of monitor container name
	MonitorContainerName = "maya-volume-exporter"
	// MgmtContainerName is the container name of cstor volume mgmt side car
	MgmtContainerName = "cstor-volume-mgmt"
)

func getDeployLabels(pvName, pvcName string) map[string]string {
	return map[string]string{
		"app":                            "cstor-volume-manager",
		"openebs.io/target":              "cstor-target",
		"openebs.io/storage-engine-type": "cstor",
		"openebs.io/cas-type":            "cstor",
		"openebs.io/persistent-volume":   pvName,
		openebsPVC:                       pvcName,
		"openebs.io/version":             version.GetVersion(),
	}
}

func getDeployAnnotation() map[string]string {
	return map[string]string{
		"openebs.io/volume-monitor": "true",
		"openebs.io/volume-type":    "cstor",
	}
}

func getDeployMatchLabels(pvName string) map[string]string {
	return map[string]string{
		"app":                          "cstor-volume-manager",
		"openebs.io/target":            "cstor-target",
		"openebs.io/persistent-volume": pvName,
	}
}

func getDeployTemplateLabels(pvName, pvcName string) map[string]string {
	return map[string]string{
		"monitoring":                   "volume_exporter_prometheus",
		"app":                          "cstor-volume-manager",
		"openebs.io/target":            "cstor-target",
		"openebs.io/persistent-volume": pvName,
		openebsPVC:                     pvcName,
		"openebs.io/version":           version.GetVersion(),
	}
}

func getDeployTemplateAnnotations() map[string]string {
	return map[string]string{
		"prometheus.io/path":   "/metrics",
		"prometheus.io/port":   "9500",
		"prometheus.io/scrape": "true",
	}
}

func getDeployOwnerReference(volume *apis.CStorVolume) []metav1.OwnerReference {
	OwnerReference := []metav1.OwnerReference{
		*metav1.NewControllerRef(volume, apis.SchemeGroupVersion.WithKind("CStorVolume")),
	}
	return OwnerReference
}

// getTargetTemplateAffinity returns affinities for target deployement
func getTargetTemplateAffinity(policySpec *apis.CStorVolumePolicySpec) *corev1.Affinity {
	if policySpec.Target.PodAffinity == nil {
		return &corev1.Affinity{}
	}
	return &corev1.Affinity{
		PodAffinity: policySpec.Target.PodAffinity,
	}
}

// getDeployTolerations returns the array of toleration
// for target deployement, defaulTolerations will be return if not provided
func getDeployTolerations(policySpec *apis.CStorVolumePolicySpec) []corev1.Toleration {

	var tolerations []corev1.Toleration
	if len(policySpec.Target.Tolerations) == 0 {
		tolerations = defaulTolerations()
	} else {
		tolerations = policySpec.Target.Tolerations
	}
	return tolerations
}

func defaulTolerations() []corev1.Toleration {
	return []corev1.Toleration{
		{
			Effect:            corev1.TaintEffectNoExecute,
			Key:               "node.alpha.kubernetes.io/notReady",
			Operator:          corev1.TolerationOpExists,
			TolerationSeconds: &tolerationSeconds,
		},
		{
			Effect:            corev1.TaintEffectNoExecute,
			Key:               "node.alpha.kubernetes.io/unreachable",
			Operator:          corev1.TolerationOpExists,
			TolerationSeconds: &tolerationSeconds,
		},
		{
			Effect:            corev1.TaintEffectNoExecute,
			Key:               "node.kubernetes.io/not-ready",
			Operator:          corev1.TolerationOpExists,
			TolerationSeconds: &tolerationSeconds,
		},
		{
			Effect:            corev1.TaintEffectNoExecute,
			Key:               "node.kubernetes.io/unreachable",
			Operator:          corev1.TolerationOpExists,
			TolerationSeconds: &tolerationSeconds,
		},
	}
}

func getMonitorMounts() []corev1.VolumeMount {
	return defaultMounts
}

func getTargetMgmtMounts() []corev1.VolumeMount {
	return defaultMounts
}

// setIstgtEnvs sets the target container performance tunables env required for
// cstorvolume target deployment
func setIstgtEnvs(policySpec *apis.CStorVolumePolicySpec) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "QueueDepth",
			Value: policySpec.Target.QueueDepth,
		},
		{
			Name:  "Luworkers",
			Value: strconv.FormatInt(policySpec.Target.IOWorkers, 10),
		},
	}
}

// getDeployTemplateEnvs return the common env required for
// cstorvolume target deployment
func getDeployTemplateEnvs(cstorid string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "OPENEBS_IO_CSTOR_VOLUME_ID",
			Value: cstorid,
		},
		{
			Name:  "RESYNC_INTERVAL",
			Value: resyncInterval,
		},
		{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "OPENEBS_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}
}

// getVolumeTargetImage returns Volume target image
// retrieves the value of the environment variable named
// by the key.
func getVolumeTargetImage() string {
	image, present := os.LookupEnv("OPENEBS_IO_CSTOR_TARGET_IMAGE")
	if !present {
		image = "openebs/cstor-istgt:ci"
	}
	return image
}

// getVolumeMonitorImage returns monitor image
// retrieves the value of the environment variable named
// by the key.
func getVolumeMonitorImage() string {
	image, present := os.LookupEnv("OPENEBS_IO_VOLUME_MONITOR_IMAGE")
	if !present {
		image = "openebs/m-exporter:ci"
	}
	return image
}

// getVolumeMgmtImage returns volume mgmt side image
// retrieves the value of the environment variable named
// by the key.
func getVolumeMgmtImage() string {
	image, present := os.LookupEnv("OPENEBS_IO_CSTOR_VOLUME_MGMT_IMAGE")
	if !present {
		image = "openebs/cstor-volume-manager-amd64:ci"
	}
	return image
}

func getContainerPort(port int32) []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			ContainerPort: port,
		},
	}
}

// getResourceRequirementForCStorTarget returns resource requirement for cstor
// target container.
func getResourceRequirementForCStorTarget(policySpec *apis.CStorVolumePolicySpec) *corev1.ResourceRequirements {
	var resourceRequirements *corev1.ResourceRequirements
	if policySpec.Target.Resources == nil {
		resourceRequirements = &corev1.ResourceRequirements{
			Limits:   nil,
			Requests: nil,
		}
	} else {
		resourceRequirements = policySpec.Target.Resources
	}
	// TODO: add default values for resources if both are nil
	return resourceRequirements
}

// getAuxResourceRequirement returns resource requirement for cstor target side car containers.
func getAuxResourceRequirement(policySpec *apis.CStorVolumePolicySpec) *corev1.ResourceRequirements {
	var auxResourceRequirements *corev1.ResourceRequirements
	if policySpec.Target.AuxResources == nil {
		auxResourceRequirements = &corev1.ResourceRequirements{
			Limits:   nil,
			Requests: nil,
		}
	} else {
		auxResourceRequirements = policySpec.Target.AuxResources
	}
	// TODO: add default values for resources if both are nil
	return auxResourceRequirements
}

func getPriorityClass(policySpec *apis.CStorVolumePolicySpec) string {
	return policySpec.Target.PriorityClassName
}

// getOrCreateCStorTargetDeployment get or create the cstor target deployment
// for a given cstorvolume.
func (c *CVCController) getOrCreateCStorTargetDeployment(
	vol *apis.CStorVolume,
	policySpec *apis.CStorVolumePolicySpec,
) (*appsv1.Deployment, error) {

	deployObj, err := c.kubeclientset.AppsV1().Deployments(openebsNamespace).
		Get(context.TODO(), vol.Name+"-target", metav1.GetOptions{})

	if err != nil && !k8serror.IsNotFound(err) {
		return nil, errors.Wrapf(
			err,
			"failed to get cstorvolume target {%v}",
			deployObj.Name,
		)
	}

	if k8serror.IsNotFound(err) {
		deployObj, err = c.BuildTargetDeployment(vol, policySpec)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to build deployment object")
		}

		deployObj, err = c.kubeclientset.AppsV1().Deployments(openebsNamespace).Create(context.TODO(), deployObj, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create deployment object")
		}
	}
	return deployObj, nil
}

// BuildTargetDeployment builds the target deploytment object for a given volume
// and policy
func (c *CVCController) BuildTargetDeployment(
	vol *apis.CStorVolume,
	policySpec *apis.CStorVolumePolicySpec,
) (*appsv1.Deployment, error) {

	deployObj := deploy.NewDeployment().
		WithName(vol.Name + "-target").
		WithLabelsNew(getDeployLabels(vol.Name, vol.GetLabels()[openebsPVC])).
		WithAnnotationsNew(getDeployAnnotation()).
		WithOwnerReferenceNew(getDeployOwnerReference(vol)).
		WithReplicas(&deployreplicas).
		WithStrategyType(
			appsv1.RecreateDeploymentStrategyType,
		).
		WithSelectorMatchLabelsNew(getDeployMatchLabels(vol.Name)).
		WithPodTemplateSpec(
			apicore.NewPodTemplateSpec().
				WithLabelsNew(getDeployTemplateLabels(vol.Name, vol.GetLabels()[openebsPVC])).
				WithAnnotationsNew(getDeployTemplateAnnotations()).
				WithServiceAccountName(util.GetServiceAccountName()).
				WithAffinity(getTargetTemplateAffinity(policySpec)).
				WithPriorityClassName(getPriorityClass(policySpec)).
				WithNodeSelectorByValue(policySpec.Target.NodeSelector).
				WithTolerationsNew(getDeployTolerations(policySpec)...).
				WithImagePullSecrets(apicore.GetImagePullSecrets(util.GetOpenEBSImagePullSecrets())).
				WithContainers(
					apicore.NewContainer().
						WithImage(getVolumeTargetImage()).
						WithName(TargetContainerName).
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithPortsNew(getContainerPort(3260)).
						WithEnvsNew(setIstgtEnvs(policySpec)).
						WithResourcesByRef(getResourceRequirementForCStorTarget(policySpec)).
						WithPrivilegedSecurityContext(&privileged).
						WithVolumeMountsNew(getTargetMgmtMounts()),
					apicore.NewContainer().
						WithImage(getVolumeMonitorImage()).
						WithName(MonitorContainerName).
						WithCommandNew([]string{"maya-exporter"}).
						WithArgumentsNew([]string{"-e=cstor"}).
						WithResourcesByRef(getAuxResourceRequirement(policySpec)).
						WithPortsNew(getContainerPort(9500)).
						WithVolumeMountsNew(getMonitorMounts()),
					apicore.NewContainer().
						WithImage(getVolumeMgmtImage()).
						WithName(MgmtContainerName).
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithPortsNew(getContainerPort(80)).
						WithEnvsNew(getDeployTemplateEnvs(string(vol.UID))).
						WithResourcesByRef(getAuxResourceRequirement(policySpec)).
						WithPrivilegedSecurityContext(&privileged).
						WithVolumeMountsNew(getTargetMgmtMounts()),
				).
				WithVolumes(
					apicore.NewVolume().
						WithName("sockfile").
						WithEmptyDir(&corev1.EmptyDirVolumeSource{}),
					apicore.NewVolume().
						WithName("conf").
						WithEmptyDir(&corev1.EmptyDirVolumeSource{}),
					apicore.NewVolume().
						WithName("storagepath").
						WithHostPathAndType(
							util.GetOpenebsBaseDirPath()+"/cstor-target/"+vol.Name,
							&hostpathType,
						),
				),
		).
		Build()
	return deployObj, nil
}
