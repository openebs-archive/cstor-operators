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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PoolType is a label for the pool type of a cStor pool.
type PoolType string

// These are the valid pool types of cStor Pool.
const (
	// PoolStriped is the striped raid group.
	PoolStriped PoolType = "stripe"
	// PoolMirrored is the mirror raid group.
	PoolMirrored PoolType = "mirror"
	// PoolRaidz is the raidz raid group.
	PoolRaidz PoolType = "raidz"
	// PoolRaidz2 is the raidz2 raid group.
	PoolRaidz2 PoolType = "raidz2"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=cstorpoolcluster

// CStorPoolCluster describes a CStorPoolCluster custom resource.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cspc
// +kubebuilder:printcolumn:name="HealthyInstances",type=integer,JSONPath=`.status.healthyInstances`,description="The number of healthy cStorPoolInstances"
// +kubebuilder:printcolumn:name="ProvisionedInstances",type=integer,JSONPath=`.status.provisionedInstances`,description="The number of provisioned cStorPoolInstances"
// +kubebuilder:printcolumn:name="DesiredInstances",type=integer,JSONPath=`.status.desiredInstances`,description="The number of desired cStorPoolInstances"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Age of CStorPoolCluster"
type CStorPoolCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CStorPoolClusterSpec   `json:"spec"`
	Status            CStorPoolClusterStatus `json:"status,omitempty"`
	VersionDetails    VersionDetails         `json:"versionDetails,omitempty"`
}

// CStorPoolClusterSpec is the spec for a CStorPoolClusterSpec resource
type CStorPoolClusterSpec struct {
	// Pools is the spec for pools for various nodes
	// where it should be created.
	Pools []PoolSpec `json:"pools,omitempty"`
	// DefaultResources are the compute resources required by the cstor-pool
	// container.
	// If the resources at PoolConfig is not specified, this is written
	// to CSPI PoolConfig.
	// +nullable
	DefaultResources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// AuxResources are the compute resources required by the cstor-pool pod
	// side car containers.
	// +nullable
	DefaultAuxResources *corev1.ResourceRequirements `json:"auxResources,omitempty"`
	// Tolerations, if specified, are the pool pod's tolerations
	// If tolerations at PoolConfig is empty, this is written to
	// CSPI PoolConfig.
	// +nullable
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// DefaultPriorityClassName if specified applies to all the pool pods
	// in the pool spec if the priorityClass at the pool level is
	// not specified.
	DefaultPriorityClassName string `json:"priorityClassName,omitempty"`
}

//PoolSpec is the spec for pool on node where it should be created.
type PoolSpec struct {
	// NodeSelector is the labels that will be used to select
	// a node for pool provisioning.
	// Required field
	NodeSelector map[string]string `json:"nodeSelector"`
	// DataRaidGroups is the raid group configuration for the given pool.
	DataRaidGroups []RaidGroup `json:"dataRaidGroups"`
	// WriteCacheRaidGroups is the write cache raid group.
	// +nullable
	WriteCacheRaidGroups []RaidGroup `json:"writeCacheRaidGroups,omitempty"`
	// PoolConfig is the default pool config that applies to the
	// pool on node.
	PoolConfig PoolConfig `json:"poolConfig,omitempty"`
}

// PoolConfig is the default pool config that applies to the
// pool on node.
type PoolConfig struct {
	// DataRaidGroupType is the  raid type.
	DataRaidGroupType string `json:"dataRaidGroupType"`

	// WriteCacheGroupType is the write cache raid type.
	WriteCacheGroupType string `json:"writeCacheGroupType,omitempty"`

	// ThickProvision to enable thick provisioning
	// Optional -- defaults to false
	ThickProvision bool `json:"thickProvision,omitempty"`
	// Compression to enable compression
	// Optional -- defaults to off
	// Possible values : lz, off
	Compression string `json:"compression,omitempty"`
	// Resources are the compute resources required by the cstor-pool
	// container.
	// +nullable
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// AuxResources are the compute resources required by the cstor-pool pod
	// side car containers.
	// +nullable
	AuxResources *corev1.ResourceRequirements `json:"auxResources,omitempty"`
	// Tolerations, if specified, the pool pod's tolerations.
	// +nullable
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// PriorityClassName if specified applies to this pool pod
	// If left empty, DefaultPriorityClassName is applied.
	// (See CStorPoolClusterSpec.DefaultPriorityClassName)
	// If both are empty, not priority class is applied.
	// +nullable
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// ROThresholdLimit is threshold(percentage base) limit
	// for pool read only mode. If ROThresholdLimit(%) amount
	// of pool storage is reached then pool will set to readonly.
	// NOTE:
	// 1. If ROThresholdLimit is set to 100 then entire
	//    pool storage will be used by default it will be set to 85%.
	// 2. ROThresholdLimit value will be 0 <= ROThresholdLimit <= 100.
	// +kubebuilder:validation:Optional
	// +nullable
	ROThresholdLimit *int `json:"roThresholdLimit,omitempty"` //optional
}

// RaidGroup contains the details of a raid group for the pool
type RaidGroup struct {
	CStorPoolInstanceBlockDevices []CStorPoolInstanceBlockDevice `json:"blockDevices"`
}

// CStorPoolInstanceBlockDevice contains the details of block devices that
// constitutes a raid group.
type CStorPoolInstanceBlockDevice struct {
	// BlockDeviceName is the name of the block device.
	BlockDeviceName string `json:"blockDeviceName"`
	// Capacity is the capacity of the block device.
	// It is system generated
	Capacity uint64 `json:"capacity,omitempty"`
	// DevLink is the dev link for block devices
	DevLink string `json:"devLink,omitempty"`
}

// CStorPoolClusterStatus represents the latest available observations of a CSPC's current state.
type CStorPoolClusterStatus struct {
	// ProvisionedInstances is the the number of CSPI present at the current state.
	// +nullable
	ProvisionedInstances int32 `json:"provisionedInstances,omitempty"`

	// DesiredInstances is the number of CSPI(s) that should be provisioned.
	// +nullable
	DesiredInstances int32 `json:"desiredInstances,omitempty"`

	// HealthyInstances is the number of CSPI(s) that are healthy.
	// +nullable
	HealthyInstances int32 `json:"healthyInstances,omitempty"`

	// Current state of CSPC.
	// +nullable
	Conditions []CStorPoolClusterCondition `json:"conditions,omitempty"`
}

type CSPCConditionType string

// These are valid conditions of a cspc.
const (
	// PoolManagerAvailable means the PoolManagerAvailable deployment is available, ie. at least the minimum available
	// replicas required are up and running and in ready state.
	PoolManagerAvailable CSPCConditionType = "PoolManagerAvailable"
)

// CStorPoolClusterCondition describes the state of a CSPC at a certain point.
type CStorPoolClusterCondition struct {
	// Type of CSPC condition.
	Type CSPCConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=cstorpoolclusters

// CStorPoolClusterList is a list of CStorPoolCluster resources
type CStorPoolClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CStorPoolCluster `json:"items"`
}
