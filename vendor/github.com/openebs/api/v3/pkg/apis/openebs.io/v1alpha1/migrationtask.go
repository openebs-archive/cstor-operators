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
// +resource:path=migrationtask
// +k8s:openapi-gen=true

// MigrationTask represents an migration task
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=mtask
type MigrationTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec i.e. specifications of the MigrationTask
	Spec MigrationTaskSpec `json:"spec"`
	// Status of MigrationTask
	Status MigrationTaskStatus `json:"status,omitempty"`
}

// MigrationTaskSpec is the properties of an migration task
type MigrationTaskSpec struct {
	MigrateResource `json:",inline"`
}

// MigrateResource is the type of resource which is to be migrated.
// Exactly one of its members must be set.
type MigrateResource struct {
	// MigrateCStorVolume contains the details of the cstor volume to be migrated
	MigrateCStorVolume *MigrateCStorVolume `json:"cstorVolume,omitempty"`
	// MigrateCStorPool contains the details of the cstor pool to be migrated
	MigrateCStorPool *MigrateCStorPool `json:"cstorPool,omitempty"`
}

// MigrateCStorVolume is the ResourceType for cstor volume
type MigrateCStorVolume struct {
	// PVName contains the name of the pv associated with the cstor volume to be migrated
	PVName string `json:"pvName,omitempty"`
}

// MigrateCStorPool is the ResourceType for cstor pool cluster
type MigrateCStorPool struct {
	// SPCName contains the name of the storage pool claim to be migrated
	SPCName string `json:"spcName,omitempty"`
	// If a CSPC with the same name as SPC already exists then we can rename
	// SPC during migration using Rename
	Rename string `json:"rename,omitempty"`
}

// MigrationTaskStatus provides status of a migrationTask
type MigrationTaskStatus struct {
	// Phase indicates if a migrationTask is started, success or errored
	Phase MigratePhase `json:"phase,omitempty"`
	// StartTime of Migrate
	StartTime metav1.Time `json:"startTime,omitempty"`
	// CompletedTime of Migrate
	CompletedTime metav1.Time `json:"completedTime,omitempty"`
	// MigrationDetailedStatuses contains the list of statuses of each step
	MigrationDetailedStatuses []MigrationDetailedStatuses `json:"migrationDetailedStatuses,omitempty"`
	// Retries is the number of times the job attempted to migration the resource
	Retries int `json:"retries,omitempty"`
}

// MigrationDetailedStatuses represents the latest available observations
// of a MigrationTask current state.
type MigrationDetailedStatuses struct {
	Step string `json:"step,omitempty"`
	// StartTime of a MigrateStep
	StartTime metav1.Time `json:"startTime,omitempty"`
	// LastUpdatedTime of a MigrateStep
	LastUpdatedTime metav1.Time `json:"lastUpdatedAt,omitempty"`
	// Phase indicates if the MigrateStep is waiting, errored or completed.
	Phase StepPhase `json:"phase,omitempty"`
	// A human-readable message indicating details about why the migrationStep
	// is in this state
	Message string `json:"message,omitempty"`
	// Reason is a brief CamelCase string that describes any failure and is meant
	// for machine parsing and tidy display in the CLI
	Reason string `json:"reason,omitempty"`
}

// MigratePhase defines phase of a MigrationTask
type MigratePhase string

const (
	// MigrateStarted - used for Migrates that are Started
	MigrateStarted MigratePhase = "Started"
	// MigrateSuccess - used for Migrates that are not available
	MigrateSuccess MigratePhase = "Success"
	// MigrateError - used for Migrates that Error for some reason
	MigrateError MigratePhase = "Error"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=migrationtasks
// +k8s:openapi-gen=true

// MigrationTaskList is a list of MigrationTask resources
type MigrationTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items are the list of migration task items
	Items []MigrationTask `json:"items"`
}
