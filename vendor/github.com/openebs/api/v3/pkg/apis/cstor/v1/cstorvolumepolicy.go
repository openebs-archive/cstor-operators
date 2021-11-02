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

// CStorVolumePolicy describes a configuration required for cstor volume
// resources
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cvp
type CStorVolumePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines a configuration info of a cstor volume required
	// to provisione cstor volume resources
	Spec   CStorVolumePolicySpec   `json:"spec"`
	Status CStorVolumePolicyStatus `json:"status,omitempty"`
}

// CStorVolumePolicySpec ...
type CStorVolumePolicySpec struct {
	// replicaAffinity is set to true then volume replica resources need to be
	// distributed across the pool instances
	Provision Provision `json:"provision,omitempty"`

	// TargetSpec represents configuration related to cstor target and its resources
	Target TargetSpec `json:"target,omitempty"`

	// ReplicaSpec represents configuration related to replicas resources
	Replica ReplicaSpec `json:"replica,omitempty"`

	// ReplicaPoolInfo holds the pool information of volume replicas.
	// Ex: If volume is provisioned on which CStor pool volume replicas exist
	ReplicaPoolInfo []ReplicaPoolInfo `json:"replicaPoolInfo,omitempty"`
}

// TargetSpec represents configuration related to cstor target and its resources
type TargetSpec struct {
	// QueueDepth sets the queue size at iSCSI target which limits the
	// ongoing IO count from client
	QueueDepth string `json:"queueDepth,omitempty"`

	// IOWorkers sets the number of threads that are working on above queue
	IOWorkers int64 `json:"luWorkers,omitempty"`

	// Monitor enables or disables the target exporter sidecar
	Monitor bool `json:"monitor,omitempty"`

	// ReplicationFactor represents maximum number of replicas
	// that are allowed to connect to the target
	ReplicationFactor int64 `json:"replicationFactor,omitempty"`

	// Resources are the compute resources required by the cstor-target
	// container.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// AuxResources are the compute resources required by the cstor-target pod
	// side car containers.
	AuxResources *corev1.ResourceRequirements `json:"auxResources,omitempty"`

	// Tolerations, if specified, are the target pod's tolerations
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// PodAffinity if specified, are the target pod's affinities
	PodAffinity *corev1.PodAffinity `json:"affinity,omitempty"`

	// NodeSelector is the labels that will be used to select
	// a node for target pod scheduleing
	// Required field
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// PriorityClassName if specified applies to this target pod
	// If left empty, no priority class is applied.
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// ReplicaSpec represents configuration related to replicas resources
type ReplicaSpec struct {
	// IOWorkers represents number of threads that executes client IOs
	IOWorkers string `json:"zvolWorkers,omitempty"`
	// Controls the compression algorithm used for this volumes
	// examples: on|off|gzip|gzip-N|lz4|lzjb|zle
	//
	// Setting compression to "on" indicates that the current default compression
	// algorithm should be used.The default balances compression and decompression
	// speed, with compression ratio and is expected to work well on a wide variety
	// of workloads. Unlike all other set‐tings for this property, on does not
	// select a fixed compression type.  As new compression algorithms are added
	// to ZFS and enabled on a pool, the default compression algorithm may change.
	// The current default compression algorithm is either lzjb or, if the
	// `lz4_compress feature is enabled, lz4.

	// The lz4 compression algorithm is a high-performance replacement for the lzjb
	// algorithm. It features significantly faster compression and decompression,
	// as well as a moderately higher compression ratio than lzjb, but can only
	// be used on pools with the lz4_compress

	// feature set to enabled.  See zpool-features(5) for details on ZFS feature
	// flags and the lz4_compress feature.

	// The lzjb compression algorithm is optimized for performance while providing
	// decent data compression.

	// The gzip compression algorithm uses the same compression as the gzip(1)
	// command.  You can specify the gzip level by using the value gzip-N,
	// where N is an integer from 1 (fastest) to 9 (best compression ratio).
	// Currently, gzip is equivalent to gzip-6 (which is also the default for gzip(1)).

	// The zle compression algorithm compresses runs of zeros.
	Compression string `json:"compression,omitempty"`
}

// Provision represents different provisioning policy for cstor volumes
type Provision struct {
	// replicaAffinity is set to true then volume replica resources need to be
	// distributed across the cstor pool instances based on the given topology
	ReplicaAffinity bool `json:"replicaAffinity"`
	// BlockSize is the logical block size in multiple of 512 bytes
	// BlockSize specifies the block size of the volume. The blocksize
	// cannot be changed once the volume has been written, so it should be
	// set at volume creation time. The default blocksize for volumes is 4 Kbytes.
	// Any power of 2 from 512 bytes to 128 Kbytes is valid.
	BlockSize uint32 `json:"blockSize,omitempty"`
}

// ReplicaPoolInfo represents the pool information of volume replica
type ReplicaPoolInfo struct {
	// PoolName represents the pool name where volume replica exists
	PoolName string `json:"poolName"`
	// UID also can be added
}

// CStorVolumePolicyStatus is for handling status of CstorVolumePolicy
type CStorVolumePolicyStatus struct {
	Phase string `json:"phase,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CStorVolumePolicyList is a list of CStorVolumePolicy resources
type CStorVolumePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CStorVolumePolicy `json:"items"`
}
