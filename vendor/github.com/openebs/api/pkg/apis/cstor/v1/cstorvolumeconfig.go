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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CStorVolumeConfig describes a cstor volume config resource created as
// custom resource. CStorVolumeConfig is a request for creating cstor volume
// related resources like deployment, svc etc.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cvc
// +kubebuilder:printcolumn:name="Capacity",type=string,JSONPath=`.status.capacity.storage`,description="Identifies the volume capacity"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`,description="Identifies the volume provisioning status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Age of CStorVolumeReplica"
type CStorVolumeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines a specification of a cstor volume config required
	// to provisione cstor volume resources
	Spec CStorVolumeConfigSpec `json:"spec"`

	// Publish contains info related to attachment of a volume to a node.
	// i.e. NodeId etc.
	Publish CStorVolumeConfigPublish `json:"publish,omitempty"`

	// Status represents the current information/status for the cstor volume
	// config, populated by the controller.
	Status         CStorVolumeConfigStatus `json:"status"`
	VersionDetails VersionDetails          `json:"versionDetails"`
}

// CStorVolumeConfigSpec is the spec for a CStorVolumeConfig resource
type CStorVolumeConfigSpec struct {
	// Capacity represents the actual resources of the underlying
	// cstor volume.
	Capacity corev1.ResourceList `json:"capacity"`
	// CStorVolumeRef has the information about where CstorVolumeClaim
	// is created from.
	CStorVolumeRef *corev1.ObjectReference `json:"cstorVolumeRef,omitempty"`
	// CStorVolumeSource contains the source volumeName@snapShotname
	// combaination.  This will be filled only if it is a clone creation.
	CStorVolumeSource string `json:"cstorVolumeSource,omitempty"`
	// Provision represents the initial volume configuration for the underlying
	// cstor volume based on the persistent volume request by user. Provision
	// properties are immutable
	Provision VolumeProvision `json:"provision"`
	// Policy contains volume specific required policies target and replicas
	Policy CStorVolumePolicySpec `json:"policy"`
}

type VolumeProvision struct {
	// Capacity represents initial capacity of volume replica required during
	// volume clone operations to maintain some metadata info related to child
	// resources like snapshot, cloned volumes.
	Capacity corev1.ResourceList `json:"capacity"`
	// ReplicaCount represents initial cstor volume replica count, its will not
	// be updated later on based on scale up/down operations, only readonly
	// operations and validations.
	ReplicaCount int `json:"replicaCount"`
}

// CStorVolumeConfigPublish contains info related to attachment of a volume to a node.
// i.e. NodeId etc.
type CStorVolumeConfigPublish struct {
	// NodeID contains publish info related to attachment of a volume to a node.
	NodeID string `json:"nodeId,omitempty"`
}

// CStorVolumeConfigPhase represents the current phase of CStorVolumeConfig.
type CStorVolumeConfigPhase string

const (
	//CStorVolumeConfigPhasePending indicates that the cvc is still waiting for
	//the cstorvolume to be created and bound
	CStorVolumeConfigPhasePending CStorVolumeConfigPhase = "Pending"

	//CStorVolumeConfigPhaseBound indiacates that the cstorvolume has been
	//provisioned and bound to the cstor volume config
	CStorVolumeConfigPhaseBound CStorVolumeConfigPhase = "Bound"

	//CStorVolumeConfigPhaseFailed indiacates that the cstorvolume provisioning
	//has failed
	CStorVolumeConfigPhaseFailed CStorVolumeConfigPhase = "Failed"
)

// CStorVolumeConfigStatus is for handling status of CstorVolume Claim.
// defines the observed state of CStorVolumeConfig
type CStorVolumeConfigStatus struct {
	// Phase represents the current phase of CStorVolumeConfig.
	Phase CStorVolumeConfigPhase `json:"phase,omitempty"`

	// PoolInfo represents current pool names where volume replicas exists
	PoolInfo []string `json:"poolInfo,omitempty"`

	// Capacity the actual resources of the underlying volume.
	Capacity corev1.ResourceList `json:"capacity,omitempty"`

	Conditions []CStorVolumeConfigCondition `json:"condition,omitempty"`
}

// CStorVolumeConfigCondition contains details about state of cstor volume
type CStorVolumeConfigCondition struct {
	// Current Condition of cstor volume config. If underlying persistent volume is being
	// resized then the Condition will be set to 'ResizeStarted' etc
	Type CStorVolumeConfigConditionType `json:"type"`
	// Last time we probed the condition.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Reason is a brief CamelCase string that describes any failure
	Reason string `json:"reason"`
	// Human-readable message indicating details about last transition.
	Message string `json:"message"`
}

// CStorVolumeConfigConditionType is a valid value of CstorVolumeConfigCondition.Type
type CStorVolumeConfigConditionType string

// These constants are CVC condition types related to resize operation.
const (
	// CStorVolumeConfigResizePending ...
	CStorVolumeConfigResizing CStorVolumeConfigConditionType = "Resizing"
	// CStorVolumeConfigResizeFailed ...
	CStorVolumeConfigResizeFailed CStorVolumeConfigConditionType = "VolumeResizeFailed"
	// CStorVolumeConfigResizeSuccess ...
	CStorVolumeConfigResizeSuccess CStorVolumeConfigConditionType = "VolumeResizeSuccessful"
	// CStorVolumeConfigResizePending ...
	CStorVolumeConfigResizePending CStorVolumeConfigConditionType = "VolumeResizePending"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// CStorVolumeConfigList is a list of CStorVolumeConfig resources
type CStorVolumeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CStorVolumeConfig `json:"items"`
}
