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

package v1

import (
	"github.com/openebs/api/v3/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

func NewCStorPoolCluster() *CStorPoolCluster {
	return &CStorPoolCluster{}
}

// WithName sets the Name field of cspc with provided value.
func (cspc *CStorPoolCluster) WithName(name string) *CStorPoolCluster {
	cspc.Name = name
	return cspc
}

// WithNamespace sets the Namespace field of cspc provided arguments
func (cspc *CStorPoolCluster) WithNamespace(namespace string) *CStorPoolCluster {
	cspc.Namespace = namespace
	return cspc
}

// WithAnnotationsNew sets the Annotations field of cspc with provided arguments
func (cspc *CStorPoolCluster) WithAnnotationsNew(annotations map[string]string) *CStorPoolCluster {
	cspc.Annotations = make(map[string]string)
	for key, value := range annotations {
		cspc.Annotations[key] = value
	}
	return cspc
}

// WithAnnotations appends or overwrites existing Annotations
// values of cspc with provided arguments
func (cspc *CStorPoolCluster) WithAnnotations(annotations map[string]string) *CStorPoolCluster {

	if cspc.Annotations == nil {
		return cspc.WithAnnotationsNew(annotations)
	}
	for key, value := range annotations {
		cspc.Annotations[key] = value
	}
	return cspc
}

// WithLabelsNew sets the Labels field of cspc with provided arguments
func (cspc *CStorPoolCluster) WithLabelsNew(labels map[string]string) *CStorPoolCluster {
	cspc.Labels = make(map[string]string)
	for key, value := range labels {
		cspc.Labels[key] = value
	}
	return cspc
}

// WithLabels appends or overwrites existing Labels
// values of cspc with provided arguments
func (cspc *CStorPoolCluster) WithLabels(labels map[string]string) *CStorPoolCluster {
	if cspc.Labels == nil {
		return cspc.WithLabelsNew(labels)
	}
	for key, value := range labels {
		cspc.Labels[key] = value
	}
	return cspc
}

// WithFinalizer sets the finalizer field in the CSPC
func (cspc *CStorPoolCluster) WithFinalizer(finalizers ...string) *CStorPoolCluster {
	cspc.Finalizers = append(cspc.Finalizers, finalizers...)
	return cspc
}

// WithDefaultResource sets the DefaultResources field in the CSPC
func (cspc *CStorPoolCluster) WithDefaultResource(resources corev1.ResourceRequirements) *CStorPoolCluster {
	cspc.Spec.DefaultResources = &resources
	return cspc
}

// WithDefaultAuxResources sets the DefaultAuxResources field in the CSPC
func (cspc *CStorPoolCluster) WithDefaultAuxResources(resources corev1.ResourceRequirements) *CStorPoolCluster {
	cspc.Spec.DefaultAuxResources = &resources
	return cspc
}

// WithTolerations sets the Tolerations field in the CSPC
func (cspc *CStorPoolCluster) WithTolerations(tolerations []corev1.Toleration) *CStorPoolCluster {
	cspc.Spec.Tolerations = tolerations
	return cspc
}

// WithDefaultPriorityClassName sets the DefaultPriorityClassName field in the CSPC
func (cspc *CStorPoolCluster) WithDefaultPriorityClassName(priorityClassName string) *CStorPoolCluster {
	cspc.Spec.DefaultPriorityClassName = priorityClassName
	return cspc
}

// WithPoolSpecs sets the Pools field in the CSPC
func (cspc *CStorPoolCluster) WithPoolSpecs(pools ...PoolSpec) *CStorPoolCluster {
	cspc.Spec.Pools = append(cspc.Spec.Pools, pools...)
	return cspc
}

func NewPoolSpec() *PoolSpec {
	return &PoolSpec{}
}

// WithNodeSelector sets the NodeSelector field in poolSpec
func (ps *PoolSpec) WithNodeSelector(nodeSelector map[string]string) *PoolSpec {
	ps.NodeSelector = nodeSelector
	return ps
}

// WithDataRaidGroups sets the DataRaidGroups field in poolSpec
func (ps *PoolSpec) WithDataRaidGroups(dataRaidGroups ...RaidGroup) *PoolSpec {
	ps.DataRaidGroups = append(ps.DataRaidGroups, dataRaidGroups...)
	return ps
}

// WithWriteCacheRaidGroups sets the WriteCacheRaidGroups field in poolSpec
func (ps *PoolSpec) WithWriteCacheRaidGroups(writeCacheRaidGroups ...RaidGroup) *PoolSpec {
	ps.WriteCacheRaidGroups = append(ps.WriteCacheRaidGroups, writeCacheRaidGroups...)
	return ps
}

// WithPoolConfig sets the PoolConfig field in poolSpec
func (ps *PoolSpec) WithPoolConfig(poolConfig PoolConfig) *PoolSpec {
	ps.PoolConfig = poolConfig
	return ps
}

func NewPoolConfig() *PoolConfig {
	return &PoolConfig{}
}

// WithDataRaidGroupType sets the DataRaidGroupType field in PoolConfig
func (pc *PoolConfig) WithDataRaidGroupType(dataRaidGroupType string) *PoolConfig {
	pc.DataRaidGroupType = dataRaidGroupType
	return pc
}

// WithWriteCacheGroupType sets the WriteCacheGroupType field in PoolConfig
func (pc *PoolConfig) WithWriteCacheGroupType(writeCacheGroupType string) *PoolConfig {
	pc.WriteCacheGroupType = writeCacheGroupType
	return pc
}

// WithThickProvision sets the ThickProvision field in PoolConfig
func (pc *PoolConfig) WithThickProvision(thickProvision bool) *PoolConfig {
	pc.ThickProvision = thickProvision
	return pc
}

// WithResources sets the Resources field in PoolConfig
func (pc *PoolConfig) WithResources(resources *corev1.ResourceRequirements) *PoolConfig {
	pc.Resources = resources
	return pc
}

// WithAuxResources sets the auxResources field in PoolConfig
func (pc *PoolConfig) WithAuxResources(auxResources *corev1.ResourceRequirements) *PoolConfig {
	pc.AuxResources = auxResources
	return pc
}

// WithTolerations sets the Tolerations field in PoolConfig
func (pc *PoolConfig) WithTolerations(tolerations []corev1.Toleration) *PoolConfig {
	pc.Tolerations = tolerations
	return pc
}

// WithPriorityClassName sets the PriorityClassName field in PoolConfig
func (pc *PoolConfig) WithPriorityClassName(priorityClassName *string) *PoolConfig {
	pc.PriorityClassName = priorityClassName
	return pc
}

// WithROThresholdLimit sets the ROThresholdLimit field in PoolConfig
func (pc *PoolConfig) WithROThresholdLimit(rOThresholdLimit *int) *PoolConfig {
	pc.ROThresholdLimit = rOThresholdLimit
	return pc
}

// NewRaidGroup returns an empty instance of raid group
func NewRaidGroup() *RaidGroup {
	return &RaidGroup{}
}

// WithROThresholdLimit sets the ROThresholdLimit field in PoolConfig
func (rg *RaidGroup) WithCStorPoolInstanceBlockDevices(cStorPoolInstanceBlockDevices ...CStorPoolInstanceBlockDevice) *RaidGroup {
	rg.CStorPoolInstanceBlockDevices = append(rg.CStorPoolInstanceBlockDevices, cStorPoolInstanceBlockDevices...)
	return rg
}

// NewCStorPoolInstanceBlockDevice returns an empty instance of CStorPoolInstanceBlockDevice
func NewCStorPoolInstanceBlockDevice() *CStorPoolInstanceBlockDevice {
	return &CStorPoolInstanceBlockDevice{}
}

// WithName sets the BlockDeviceName field in CStorPoolInstanceBlockDevice
func (cspibd *CStorPoolInstanceBlockDevice) WithName(name string) *CStorPoolInstanceBlockDevice {
	cspibd.BlockDeviceName = name
	return cspibd
}

// HasFinalizer returns true if the provided finalizer is present on the object.
func (cspc *CStorPoolCluster) HasFinalizer(finalizer string) bool {
	finalizersList := cspc.GetFinalizers()
	return util.ContainsString(finalizersList, finalizer)
}

// RemoveFinalizer removes the given finalizer from the object.
func (cspc *CStorPoolCluster) RemoveFinalizer(finalizer string) {
	cspc.Finalizers = util.RemoveString(cspc.Finalizers, finalizer)
}

// HasAnnotation return true if provided annotation
// key and value are present on the object.
func (cspc *CStorPoolCluster) HasAnnotation(key, value string) bool {
	val, ok := cspc.GetAnnotations()[key]
	if ok {
		return val == value
	}
	return false
}

// HasLabel returns true if provided label
// key and value are present on the object.
func (cspc *CStorPoolCluster) HasLabel(key, value string) bool {
	val, ok := cspc.GetLabels()[key]
	if ok {
		return val == value
	}
	return false
}

// GetBlockDevices returns list of blockdevice names exist in the raid group
func (rg RaidGroup) GetBlockDevices() []string {
	var bdNames []string
	for _, cspcBD := range rg.CStorPoolInstanceBlockDevices {
		bdNames = append(bdNames, cspcBD.BlockDeviceName)
	}
	return bdNames
}
