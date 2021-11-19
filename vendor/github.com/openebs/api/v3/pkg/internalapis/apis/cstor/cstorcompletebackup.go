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
// +resource:path=cstorcompletedbackup

// CStorCompletedBackup describes a cstor completed-backup resource created as custom resource
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=ccompletedbackup
// +kubebuilder:printcolumn:name="Volume",type=string,JSONPath=`.spec.volumeName`,description="Volume name on which backup is performed"
// +kubebuilder:printcolumn:name="Backup/Schedule",type=string,JSONPath=`.spec.backupName`,description="Name of the backup or scheduled backup"
// +kubebuilder:printcolumn:name="LastSnap",type=string,JSONPath=`.spec.lastSnapName`,description="Last successfully backup snapshot"
type CStorCompletedBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CStorCompletedBackupSpec `json:"spec"`
}

// CStorCompletedBackupSpec is the spec for a CStorBackup resource
type CStorCompletedBackupSpec struct {
	// BackupName is the name of backup or scheduled backup
	BackupName string `json:"backupName,omitempty"`

	// VolumeName is the name of volume for which this backup is destined
	VolumeName string `json:"volumeName,omitempty"`

	// SecondLastSnapName is the name of second last 'successfully' completed-backup's snapshot
	SecondLastSnapName string `json:"secondLastSnapName,omitempty"`

	// LastSnapName is the name of last completed-backup's snapshot name
	LastSnapName string `json:"lastSnapName,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=cstorcompletedbackup

// CStorCompletedBackupList is a list of cstorcompletedbackup resources
type CStorCompletedBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CStorCompletedBackup `json:"items"`
}
