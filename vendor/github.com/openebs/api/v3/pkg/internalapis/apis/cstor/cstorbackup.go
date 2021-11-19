/*
Copyright 2020 The OpenEBS Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=cstorbackup

// CStorBackup describes a cstor backup resource created as a custom resource
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cbackup
// +kubebuilder:printcolumn:name="Volume",type=string,JSONPath=`.spec.volumeName`,description="Name of the volume for which this backup is destined"
// +kubebuilder:printcolumn:name="Backup/Schedule",type=string,JSONPath=`.spec.backupName`,description="Name of the backup or scheduled backup"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status`,description="Identifies the phase of the backup"
type CStorBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CStorBackupSpec   `json:"spec"`
	Status            CStorBackupStatus `json:"status,omitempty"`
}

// CStorBackupSpec is the spec for a CStorBackup resource
type CStorBackupSpec struct {
	// BackupName is the name of the backup or scheduled backup
	BackupName string `json:"backupName"`

	// VolumeName is the name of the volume for which this backup is destined
	VolumeName string `json:"volumeName"`

	// SnapName is the name of the current backup snapshot
	SnapName string `json:"snapName"`

	// PrevSnapName is the last completed-backup's snapshot name
	PrevSnapName string `json:"prevSnapName,omitempty"`

	// BackupDest is the remote address for backup transfer
	BackupDest string `json:"backupDest,omitempty"`

	// LocalSnap is the flag to enable local snapshot only
	LocalSnap bool `json:"localSnap,omitempty"`
}

// CStorBackupStatus is a string type that represents the status of the backup
type CStorBackupStatus string

// Status written onto CStorBackup objects
const (
	// BKPCStorStatusEmpty represents backup resource is created and yet to
	// be reconcile by backup controller
	BKPCStorStatusEmpty CStorBackupStatus = ""

	// BKPCStorStatusDone , backup is completed
	BKPCStorStatusDone CStorBackupStatus = "Done"

	// BKPCStorStatusFailed , backup is failed
	BKPCStorStatusFailed CStorBackupStatus = "Failed"

	// BKPCStorStatusInit , backup is initialized
	BKPCStorStatusInit CStorBackupStatus = "Init"

	// BKPCStorStatusPending , backup is pending
	BKPCStorStatusPending CStorBackupStatus = "Pending"

	// BKPCStorStatusInProgress , backup is in progress
	BKPCStorStatusInProgress CStorBackupStatus = "InProgress"

	// BKPCStorStatusInvalid , backup operation is invalid
	BKPCStorStatusInvalid CStorBackupStatus = "Invalid"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=cstorbackup

// CStorBackupList is a list of CStorBackup resources
type CStorBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CStorBackup `json:"items"`
}
