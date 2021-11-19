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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=cstorvolume

// CStorVolume describes a cstor volume resource created as custom resource
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cv
// +kubebuilder:printcolumn:name="Capacity",type=string,JSONPath=`.status.capacity`,description="Current volume capacity"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`,description="Identifies the current health of the volume"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Age of CStorVolume"
type CStorVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CStorVolumeSpec   `json:"spec"`
	Status            CStorVolumeStatus `json:"status,omitempty"`
	VersionDetails    VersionDetails    `json:"versionDetails,omitempty"`
}

// CStorVolumeSpec is the spec for a CStorVolume resource
type CStorVolumeSpec struct {
	// Capacity represents the desired size of the underlying volume.
	Capacity resource.Quantity `json:"capacity,omitempty"`

	// TargetIP IP of the iSCSI target service
	TargetIP string `json:"targetIP,omitempty"`

	// iSCSI Target Port typically TCP ports 3260
	TargetPort string `json:"targetPort,omitempty"`

	// Target iSCSI Qualified Name.combination of nodeBase
	Iqn string `json:"iqn,omitempty"`

	// iSCSI Target Portal. The Portal is combination of IP:port (typically TCP ports 3260)
	TargetPortal string `json:"targetPortal,omitempty"`

	// ReplicationFactor represents number of volume replica created during volume
	// provisioning connect to the target
	ReplicationFactor int `json:"replicationFactor,omitempty"`

	// ConsistencyFactor is minimum number of volume replicas i.e. `RF/2 + 1`
	// has to be connected to the target for write operations. Basically more then
	// 50% of replica has to be connected to target.
	ConsistencyFactor int `json:"consistencyFactor,omitempty"`

	// DesiredReplicationFactor represents maximum number of replicas
	// that are allowed to connect to the target. Required for scale operations
	DesiredReplicationFactor int `json:"desiredReplicationFactor,omitempty"`

	//ReplicaDetails refers to the trusty replica information
	ReplicaDetails CStorVolumeReplicaDetails `json:"replicaDetails,omitempty"`
}

// ReplicaID is to hold replicaID information
type ReplicaID string

// CStorVolumePhase is to hold result of action.
type CStorVolumePhase string

// CStorVolumeStatus is for handling status of cvr.
type CStorVolumeStatus struct {
	Phase           CStorVolumePhase `json:"phase,omitempty"`
	ReplicaStatuses []ReplicaStatus  `json:"replicaStatuses,omitempty"`
	// Represents the actual capacity of the underlying volume.
	Capacity resource.Quantity `json:"capacity,omitempty"`
	// LastTransitionTime refers to the time when the phase changes
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// LastUpdateTime refers to the time when last status updated due to any
	// operations
	// +nullable
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// A human-readable message indicating details about why the volume is in this state.
	Message string `json:"message,omitempty"`
	// Current Condition of cstorvolume. If underlying persistent volume is being
	// resized then the Condition will be set to 'ResizePending'.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []CStorVolumeCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,4,rep,name=conditions"`
	// ReplicaDetails refers to the trusty replica information
	ReplicaDetails CStorVolumeReplicaDetails `json:"replicaDetails,omitempty"`
}

// CStorVolumeReplicaDetails contains trusty replica inform which will be
// updated by target
type CStorVolumeReplicaDetails struct {
	// KnownReplicas represents the replicas that target can trust to read data
	KnownReplicas map[ReplicaID]string `json:"knownReplicas,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=cstorvolume

// CStorVolumeList is a list of CStorVolume resources
type CStorVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CStorVolume `json:"items"`
}

// CVStatusResponse stores the reponse of istgt replica command output
// It may contain several volumes
type CVStatusResponse struct {
	CVStatuses []CVStatus `json:"volumeStatus"`
}

// CVStatus stores the status of a CstorVolume obtained from response
type CVStatus struct {
	Name            string          `json:"name"`
	Status          string          `json:"status"`
	ReplicaStatuses []ReplicaStatus `json:"replicaStatus"`
}

// ReplicaStatus stores the status of replicas
type ReplicaStatus struct {
	// ID is replica unique identifier
	ID string `json:"replicaId"`
	// Mode represents replica status i.e. Healthy, Degraded
	Mode string `json:"mode"`
	// Represents IO number of replica persisted on the disk
	CheckpointedIOSeq string `json:"checkpointedIOSeq"`
	// Ongoing reads I/O from target to replica
	InflightRead string `json:"inflightRead"`
	// ongoing writes I/O from target to replica
	InflightWrite string `json:"inflightWrite"`
	// Ongoing sync I/O from target to replica
	InflightSync string `json:"inflightSync"`
	// time since the replica connected to target
	UpTime int `json:"upTime"`
	// Quorum indicates wheather data wrtitten to the replica
	// is lost or exists.
	// "0" means: data has been lost( might be ephimeral case)
	// and will recostruct data from other Healthy replicas in a write-only
	// mode
	// 1 means: written data is exists on replica
	Quorum string `json:"quorum"`
}

// CStorVolumeCondition contains details about state of cstorvolume
type CStorVolumeCondition struct {
	Type   CStorVolumeConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=CStorVolumeConditionType"`
	Status ConditionStatus          `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// Last time we probed the condition.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty" protobuf:"bytes,3,opt,name=lastProbeTime"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	// Unique, this should be a short, machine understandable string that gives the reason
	// for condition's last transition. If it reports "ResizePending" that means the underlying
	// cstorvolume is being resized.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// CStorVolumeConditionType is a valid value of CStorVolumeCondition.Type
type CStorVolumeConditionType string

const (
	// CStorVolumeResizing - a user trigger resize of pvc has been started
	CStorVolumeResizing CStorVolumeConditionType = "Resizing"
)

// ConditionStatus states in which state condition is present
type ConditionStatus string

// These are valid condition statuses. "ConditionInProgress" means corresponding
// condition is inprogress. "ConditionSuccess" means corresponding condition is success
const (
	// ConditionInProgress states resize of underlying volumes are in progress
	ConditionInProgress ConditionStatus = "InProgress"
	// ConditionSuccess states resizing underlying volumes are successfull
	ConditionSuccess ConditionStatus = "Success"
)

// Status written onto CStorVolume objects.
const (
	// volume is getting initialized
	CVStatusInit CStorVolumePhase = "Init"
	// volume allows IOs and snapshot
	CVStatusHealthy CStorVolumePhase = "Healthy"
	// volume only satisfies consistency factor
	CVStatusDegraded CStorVolumePhase = "Degraded"
	// Volume is offline
	CVStatusOffline CStorVolumePhase = "Offline"
	// Error in retrieving volume details
	CVStatusError CStorVolumePhase = "Error"
	// volume controller config generation failed due to invalid parameters
	CVStatusInvalid CStorVolumePhase = "Invalid"
)
