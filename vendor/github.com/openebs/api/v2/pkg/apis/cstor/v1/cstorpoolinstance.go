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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=cstorpoolinstance

// CStorPoolInstance describes a cstor pool instance resource.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cspi
// +kubebuilder:printcolumn:name="HostName",type=string,JSONPath=`.spec.hostName`,description="Host name where cstorpool instances scheduled"
// +kubebuilder:printcolumn:name="Allocated",type=string,JSONPath=`.status.capacity.used`,description="The amount of storage space within the pool that has been physically allocated",priority=1
// +kubebuilder:printcolumn:name="Free",type=string,JSONPath=`.status.capacity.free`,description="The amount of usable free space available in the pool"
// +kubebuilder:printcolumn:name="Capacity",type=string,JSONPath=`.status.capacity.total`,description="Total amount of usable space in pool"
// +kubebuilder:printcolumn:name="ReadOnly",type=boolean,JSONPath=`.status.readOnly`,description="Identifies the pool read only mode"
// +kubebuilder:printcolumn:name="ProvisionedReplicas",type=integer,JSONPath=`.status.provisionedReplicas`,description="Represents no.of replicas present in the pool"
// +kubebuilder:printcolumn:name="HealthyReplicas",type=integer,JSONPath=`.status.healthyReplicas`,description="Represents no.of healthy replicas present in the pool"
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.poolConfig.dataRaidGroupType`,description="Represents the type of the storage pool",priority=1
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`,description="Identifies the current health of the pool"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Age of CStorPoolInstance"
type CStorPoolInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the specification of the cstorpoolinstance resource.
	Spec CStorPoolInstanceSpec `json:"spec"`
	// Status is the possible statuses of the cstorpoolinstance resource.
	Status CStorPoolInstanceStatus `json:"status,omitempty"`
	// VersionDetails is the openebs version.
	VersionDetails VersionDetails `json:"versionDetails,omitempty"`
}

// CStorPoolInstanceSpec is the spec listing fields for a CStorPoolInstance resource.
type CStorPoolInstanceSpec struct {
	// HostName is the name of kubernetes node where the pool
	// should be created.
	HostName string `json:"hostName,omitempty"`
	// NodeSelector is the labels that will be used to select
	// a node for pool provisioning.
	// Required field
	NodeSelector map[string]string `json:"nodeSelector"`
	// PoolConfig is the default pool config that applies to the
	// pool on node.
	PoolConfig PoolConfig `json:"poolConfig,omitempty"`
	// DataRaidGroups is the raid group configuration for the given pool.
	DataRaidGroups []RaidGroup `json:"dataRaidGroups"`
	// WriteCacheRaidGroups is the write cache raid group.
	// +nullable
	WriteCacheRaidGroups []RaidGroup `json:"writeCacheRaidGroups,omitempty"`
}

// CStorPoolInstancePhase is the phase for CStorPoolInstance resource.
type CStorPoolInstancePhase string

// Status written onto CStorPool and CStorVolumeReplica objects.
// Resetting state to either Init or CreateFailed need to be done with care,
// as, label clear and pool creation depends on this state.
const (
	// CStorPoolStatusEmpty ensures the create operation is to be done, if import fails.
	CStorPoolStatusEmpty CStorPoolInstancePhase = ""
	// CStorPoolStatusOnline signifies that the pool is online.
	CStorPoolStatusOnline CStorPoolInstancePhase = "ONLINE"
	// CStorPoolStatusOffline signifies that the pool is offline.
	CStorPoolStatusOffline CStorPoolInstancePhase = "OFFLINE"
	// CStorPoolStatusDegraded signifies that the pool is degraded.
	CStorPoolStatusDegraded CStorPoolInstancePhase = "DEGRADED"
	// CStorPoolStatusFaulted signifies that the pool is faulted.
	CStorPoolStatusFaulted CStorPoolInstancePhase = "FAULTED"
	// CStorPoolStatusRemoved signifies that the pool is removed.
	CStorPoolStatusRemoved CStorPoolInstancePhase = "REMOVED"
	// CStorPoolStatusUnavail signifies that the pool is not available.
	CStorPoolStatusUnavail CStorPoolInstancePhase = "UNAVAIL"
	// CStorPoolStatusError signifies that the pool status could not be fetched.
	CStorPoolStatusError CStorPoolInstancePhase = "Error"
	// CStorPoolStatusDeletionFailed ensures the resource deletion has failed.
	CStorPoolStatusDeletionFailed CStorPoolInstancePhase = "DeletionFailed"
	// CStorPoolStatusInvalid ensures invalid resource.
	CStorPoolStatusInvalid CStorPoolInstancePhase = "Invalid"
	// CStorPoolStatusErrorDuplicate ensures error due to duplicate resource.
	CStorPoolStatusErrorDuplicate CStorPoolInstancePhase = "ErrorDuplicate"
	// CStorPoolStatusPending ensures pending task for cstorpool.
	CStorPoolStatusPending CStorPoolInstancePhase = "Pending"
	// CStorPoolStatusInit is initial state of CSP, before pool creation.
	CStorPoolStatusInit CStorPoolInstancePhase = "Init"
	// CStorPoolStatusCreateFailed is state when pool creation failed
	CStorPoolStatusCreateFailed CStorPoolInstancePhase = "PoolCreationFailed"
)

// CStorPoolInstanceStatus is for handling status of pool.
type CStorPoolInstanceStatus struct {
	// Current state of CSPI with details.
	Conditions []CStorPoolInstanceCondition `json:"conditions,omitempty"`
	//  The phase of a CStorPool is a simple, high-level summary of the pool state on the
	//  node.
	Phase CStorPoolInstancePhase `json:"phase,omitempty"`
	// Capacity describes the capacity details of a cstor pool
	Capacity CStorPoolInstanceCapacity `json:"capacity,omitempty"`
	//ReadOnly if pool is readOnly or not
	ReadOnly bool `json:"readOnly"`
	// ProvisionedReplicas describes the total count of Volume Replicas
	// present in the cstor pool
	ProvisionedReplicas int32 `json:"provisionedReplicas"`
	// HealthyReplicas describes the total count of healthy Volume Replicas
	// in the cstor pool
	HealthyReplicas int32 `json:"healthyReplicas"`
}

// CStorPoolInstanceCapacity stores the pool capacity related attributes.
type CStorPoolInstanceCapacity struct {
	// Amount of physical data (and its metadata) written to pool
	// after applying compression, etc..,
	Used resource.Quantity `json:"used"`
	// Amount of usable space in the pool after excluding
	// metadata and raid parity
	Free resource.Quantity `json:"free"`
	// Sum of usable capacity in all the data raidgroups
	Total resource.Quantity `json:"total"`
	// ZFSCapacityAttributes contains advanced information about pool capacity details
	ZFS ZFSCapacityAttributes `json:"zfs"`
}

// ZFSCapacityAttributes stores the advanced information about pool capacity related
// attributes
type ZFSCapacityAttributes struct {
	// LogicalUsed is the amount of space that is "logically" consumed
	// by this pool and all its descendents. The logical space ignores
	// the effect of the compression and copies properties, giving a
	// quantity closer to the amount of data that applications see.
	// However, it does include space consumed by metadata.
	LogicalUsed resource.Quantity `json:"logicalUsed"`
}

type CStorPoolInstanceConditionType string

const (
	// CSPIPoolExpansion condition will be available when user triggers
	// pool expansion by adding blockdevice/raidgroup (or) when underlying
	// disk got expanded
	CSPIPoolExpansion CStorPoolInstanceConditionType = "PoolExpansion"
	// CSPIDiskReplacement condition will be available when user triggers
	// disk replacement by replacing the blockdevice
	CSPIDiskReplacement CStorPoolInstanceConditionType = "DiskReplacement"
	// CSPIDiskUnavailable condition will be available when one (or) more
	// disks were unavailable
	CSPIDiskUnavailable CStorPoolInstanceConditionType = "DiskUnavailable"
	// CSPIPoolLost condition will be available when unable to import the pool
	CSPIPoolLost CStorPoolInstanceConditionType = "PoolLost"
)

// CSPIConditionType describes the state of a CSPI at a certain point.
type CStorPoolInstanceCondition struct {
	// Type of CSPC condition.
	Type CStorPoolInstanceConditionType `json:"type"`
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
// +resource:path=cstorpoolinstance

// CStorPoolInstanceList is a list of CStorPoolInstance resources
type CStorPoolInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []CStorPoolInstance `json:"items"`
}
