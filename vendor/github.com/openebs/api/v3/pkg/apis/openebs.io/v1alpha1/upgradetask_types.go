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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=upgradetask
// +k8s:openapi-gen=true

// UpgradeTask represents an upgrade task
type UpgradeTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec i.e. specifications of the UpgradeTask
	Spec UpgradeTaskSpec `json:"spec"`
	// Status of UpgradeTask
	Status UpgradeTaskStatus `json:"status,omitempty"`
}

// UpgradeTaskSpec is the properties of an upgrade task
type UpgradeTaskSpec struct {
	// FromVersion is the current version of the resource.
	FromVersion string `json:"fromVersion"`
	// ToVersion is the upgraded version of the resource. It should be same
	// as the version of control plane components version.
	ToVersion string `json:"toVersion"`
	// Options contains the optional flags that can be passed during upgrade.
	Options *Options `json:"options,omitempty"`
	// ResourceSpec contains the details of the resource that has to upgraded.
	ResourceSpec `json:",inline"`
	// ImagePrefix contains the url prefix of the image url. This field is
	// optional. If not present upgrade takes the previously present ImagePrefix.
	ImagePrefix string `json:"imagePrefix,omitempty"`
	// ImageTag contains the customized tag for ToVersion if any. This field is
	// optional. If not present upgrade takes the ToVersion as the ImageTag
	ImageTag string `json:"imageTag,omitempty"`
}

// Options provides additional optional arguments
type Options struct {
	// Timeout is maximum seconds to wait at any given step in the upgrade
	Timeout int `json:"timeout,omitempty"`
}

// ResourceSpec is the type of resource which is to be upgraded.
// Exactly one of its members must be set.
type ResourceSpec struct {
	// JivaVolume contains the details of the jiva volume to be upgraded
	JivaVolume *JivaVolume `json:"jivaVolume,omitempty"`
	// CStorVolume contains the details of the cstor volume to be upgraded
	CStorVolume *CStorVolume `json:"cstorVolume,omitempty"`
	// CStorPool contains the details of the cstor pool to be upgraded
	CStorPool *CStorPool `json:"cstorPool,omitempty"`
	// StoragePoolClaim contains the details of the storage pool claim to be upgraded
	StoragePoolClaim *StoragePoolClaim `json:"storagePoolClaim,omitempty"`
	// CStorPoolInstance contains the details of the cstor pool to be upgraded
	CStorPoolInstance *CStorPoolInstance `json:"cstorPoolInstance,omitempty"`
	// CStorPoolCluster contains the details of the storage pool claim to be upgraded
	CStorPoolCluster *CStorPoolCluster `json:"cstorPoolCluster,omitempty"`
}

// JivaVolume is the ResourceType for jiva volume
type JivaVolume struct {
	// PVName contains the name of the pv associated with the jiva volume
	PVName string `json:"pvName,omitempty"`
	// Options can be used to change the default behaviour of upgrade
	Options *ResourceOptions `json:"options,omitempty"`
}

// CStorVolume is the ResourceType for cstor volume
type CStorVolume struct {
	// PVName contains the name of the pv associated with the cstor volume
	PVName string `json:"pvName,omitempty"`
	// Options can be used to change the default behaviour of upgrade
	Options *ResourceOptions `json:"options,omitempty"`
}

// CStorPoolCluster is the ResourceType for cstor pool cluster
type CStorPoolCluster struct {
	// CSPCName contains the name of the storage pool claim to be upgraded
	CSPCName string `json:"cspcName,omitempty"`
	// Options can be used to change the default behaviour of upgrade
	Options *ResourceOptions `json:"options,omitempty"`
}

// CStorPool is the ResourceType for cstor pool
type CStorPool struct {
	// PoolName contains the name of the cstor pool to be upgraded
	PoolName string `json:"poolName,omitempty"`
	// Options can be used to change the default behaviour of upgrade
	Options *ResourceOptions `json:"options,omitempty"`
}

// StoragePoolClaim is the ResourceType for storage pool claim
type StoragePoolClaim struct {
	// SPCName contains the name of the storage pool claim to be upgraded
	SPCName string `json:"spcName,omitempty"`
	// Options can be used to change the default behaviour of upgrade
	Options *ResourceOptions `json:"options,omitempty"`
}

// CStorPoolInstance is the ResourceType for cstor pool cluster
type CStorPoolInstance struct {
	// CSPCName contains the name of the storage pool claim to be upgraded
	CSPIName string `json:"cspiName,omitempty"`
	// Options can be used to change the default behaviour of upgrade
	Options *ResourceOptions `json:"options,omitempty"`
}

// ResourceOptions provides additional options for a particular resource
type ResourceOptions struct {
	// IgnoreStepsOnError allows to ignore steps which failed
	IgnoreStepsOnError []string `json:"ignoreStepsOnError,omitempty"`
}

// UpgradeTaskStatus provides status of a upgradeTask
type UpgradeTaskStatus struct {
	// Phase indicates if a upgradeTask is started, success or errored
	Phase UpgradePhase `json:"phase,omitempty"`
	// StartTime of Upgrade
	// +nullable
	StartTime metav1.Time `json:"startTime,omitempty"`
	// CompletedTime of Upgrade
	// +nullable
	CompletedTime metav1.Time `json:"completedTime,omitempty"`
	// UpgradeDetailedStatuses contains the list of statuses of each step
	UpgradeDetailedStatuses []UpgradeDetailedStatuses `json:"upgradeDetailedStatuses,omitempty"`
	// Retries is the number of times the job attempted to upgrade the resource
	Retries int `json:"retries,omitempty"`
}

// UpgradeDetailedStatuses represents the latest available observations
// of a UpgradeTask current state.
type UpgradeDetailedStatuses struct {
	Step UpgradeStep `json:"step,omitempty"`
	// StartTime of a UpgradeStep
	// +nullable
	StartTime metav1.Time `json:"startTime,omitempty"`
	// LastUpdatedTime of a UpgradeStep
	// +nullable
	LastUpdatedTime metav1.Time `json:"lastUpdatedAt,omitempty"`
	// Status of a UpgradeStep
	Status `json:",inline"`
}

// Status represents the state of the step performed during the upgrade.
type Status struct {
	// Phase indicates if the UpgradeStep is waiting, errored or completed.
	Phase StepPhase `json:"phase,omitempty"`
	// A human-readable message indicating details about why the upgradeStep
	// is in this state
	Message string `json:"message,omitempty"`
	// Reason is a brief CamelCase string that describes any failure and is meant
	// for machine parsing and tidy display in the CLI
	Reason string `json:"reason,omitempty"`
}

// UpgradeStep is the current step being performed for a particular resource upgrade
type UpgradeStep string

const (
	// PreUpgrade is the step to verify resource before upgrade
	PreUpgrade UpgradeStep = "PRE_UPGRADE"
	// TargetUpgrade is the step to upgrade Target depoyment of resource
	TargetUpgrade UpgradeStep = "TARGET_UPGRADE"
	// ReplicaUpgrade is the step to upgrade replica deployment of resource
	ReplicaUpgrade UpgradeStep = "REPLICA_UPGRADE"
	// Verify is the step to verify the upgrade
	Verify UpgradeStep = "VERIFY"
	// Rollback is the step to rollback to previous version if upgrade fails
	Rollback UpgradeStep = "ROLLBACK"
	// PoolInstanceUpgrade is the step to upgrade a pool (CSP or CSPI and it's deployment)
	PoolInstanceUpgrade UpgradeStep = "POOL_INSTANCE_UPGRADE"
)

// StepPhase defines the phase of a UpgradeStep
type StepPhase string

const (
	// StepWaiting - used for upgrade step that not yet complete
	StepWaiting StepPhase = "Waiting"
	// StepErrored - used for upgrade step that failed
	StepErrored StepPhase = "Errored"
	// StepCompleted - used for upgrade step that completed successfully
	StepCompleted StepPhase = "Completed"
)

// UpgradePhase defines phase of a UpgradeTask
type UpgradePhase string

const (
	// UpgradeStarted - used for Upgrades that are Started
	UpgradeStarted UpgradePhase = "Started"
	// UpgradeSuccess - used for Upgrades that are not available
	UpgradeSuccess UpgradePhase = "Success"
	// UpgradeError - used for Upgrades that Error for some reason
	UpgradeError UpgradePhase = "Error"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=upgradetasks
// +k8s:openapi-gen=true

// UpgradeTaskList is a list of UpgradeTask resources
type UpgradeTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items are the list of upgrade task items
	Items []UpgradeTask `json:"items"`
}
