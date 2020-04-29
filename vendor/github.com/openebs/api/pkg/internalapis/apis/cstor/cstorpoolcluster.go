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

package cstor

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
// +k8s:openapi-gen=true
// +resource:path=cstorpoolcluster

// CStorPoolCluster describes a CStorPoolCluster custom resource.
type CStorPoolCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CStorPoolClusterSpec   `json:"spec"`
	Status            CStorPoolClusterStatus `json:"status"`
	VersionDetails    VersionDetails         `json:"versionDetails"`
}

// CStorPoolClusterSpec is the spec for a CStorPoolClusterSpec resource
type CStorPoolClusterSpec struct {
	// Pools is the spec for pools for various nodes
	// where it should be created.
	Pools []PoolSpec `json:"pools"`
	// DefaultResources are the compute resources required by the cstor-pool
	// container.
	// If the resources at PoolConfig is not specified, this is written
	// to CSPI PoolConfig.
	DefaultResources *corev1.ResourceRequirements `json:"resources"`
	// AuxResources are the compute resources required by the cstor-pool pod
	// side car containers.
	DefaultAuxResources *corev1.ResourceRequirements `json:"auxResources"`
	// Tolerations, if specified, are the pool pod's tolerations
	// If tolerations at PoolConfig is empty, this is written to
	// CSPI PoolConfig.
	Tolerations []corev1.Toleration `json:"tolerations"`

	// DefaultPriorityClassName if specified applies to all the pool pods
	// in the pool spec if the priorityClass at the pool level is
	// not specified.
	DefaultPriorityClassName string `json:"priorityClassName"`
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
	WriteCacheRaidGroups []RaidGroup `json:"writeCacheRaidGroups"`
	// PoolConfig is the default pool config that applies to the
	// pool on node.
	PoolConfig PoolConfig `json:"poolConfig"`
}

// PoolConfig is the default pool config that applies to the
// pool on node.
type PoolConfig struct {
	// DataRaidGroupType is the  raid type.
	DataRaidGroupType string `json:"dataRaidGroupType"`

	// WriteCacheGroupType is the write cache raid type.
	WriteCacheGroupType string `json:"writeCacheGroupType"`

	// ThickProvision to enable thick provisioning
	// Optional -- defaults to false
	ThickProvision bool `json:"thickProvision"`
	// Compression to enable compression
	// Optional -- defaults to off
	// Possible values : lz, off
	Compression string `json:"compression"`
	// Resources are the compute resources required by the cstor-pool
	// container.
	Resources *corev1.ResourceRequirements `json:"resources"`
	// AuxResources are the compute resources required by the cstor-pool pod
	// side car containers.
	AuxResources *corev1.ResourceRequirements `json:"auxResources"`
	// Tolerations, if specified, the pool pod's tolerations.
	Tolerations []corev1.Toleration `json:"tolerations"`

	// PriorityClassName if specified applies to this pool pod
	// If left empty, DefaultPriorityClassName is applied.
	// (See CStorPoolClusterSpec.DefaultPriorityClassName)
	// If both are empty, not priority class is applied.
	PriorityClassName *string `json:"priorityClassName"`

	// ROThresholdLimit is threshold(percentage base) limit
	// for pool read only mode. If ROThresholdLimit(%) amount
	// of pool storage is reached then pool will set to readonly.
	// NOTE:
	// 1. If ROThresholdLimit is set to 100 then entire
	//    pool storage will be used by default it will be set to 85%.
	// 2. ROThresholdLimit value will be 0 <= ROThresholdLimit <= 100.
	ROThresholdLimit *int `json:"roThresholdLimit"` //optional
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
	Capacity uint64 `json:"capacity"`
	// DevLink is the dev link for block devices
	DevLink string `json:"devLink"`
}

// CStorPoolClusterStatus represents the latest available observations of a CSPC's current state.
type CStorPoolClusterStatus struct {
	// ProvisionedInstances is the the number of CSPI present at the current state.
	ProvisionedInstances int32 `json:"provisionedInstances"`

	// DesiredInstances is the number of CSPI(s) that should be provisioned.
	DesiredInstances int32 `json:"desiredInstances"`

	// HealthyInstances is the number of CSPI(s) that are healthy.
	HealthyInstances int32 `json:"healthyInstances"`

	// Current state of CSPC.
	Conditions []CStorPoolClusterCondition `json:conditions`
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
