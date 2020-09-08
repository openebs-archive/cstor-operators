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

package types

import "sync"

const (
	// OpenEBSDisableReconcileLabelKey is the label key decides to reconcile or not
	OpenEBSDisableReconcileLabelKey = "reconcile.openebs.io/disable"
	// CStorAPIVersion is group version for cstor apis
	CStorAPIVersion = "cstor.openebs.io/v1"

	// HostNameLabelKey is label key present on kubernetes node object.
	HostNameLabelKey = "kubernetes.io/hostname"

	// CStorPoolClusterLabelKey is the CStorPoolcluster label key.
	CStorPoolClusterLabelKey = "openebs.io/cstor-pool-cluster"

	// CStorPoolInstanceLabelKey is the CStorPoolInstance label
	CStorPoolInstanceLabelKey = "openebs.io/cstor-pool-instance"

	// OpenEBSVersionLabelKey is the openebs version key.
	OpenEBSVersionLabelKey = "openebs.io/version"

	// CASTypeLabelKey is the label key to fetch storage engine for the volume
	CASTypeLabelKey = "openebs.io/cas-type"

	// PredecessorBDKey is the key to fetch the predecessor BD in case of
	// block device replacement.
	PredecessorBDLabelKey = "openebs.io/bd-predecessor"

	//PodDisruptionBudgetKey is the key used to identify the PDB
	PodDisruptionBudgetKey = "openebs.io/pod-disruption-budget"

	// VolumePolicyKey is the key to fetch name of CStorVolume Policies
	VolumePolicyKey = "openebs.io/volume-policy"

	// CStorPoolInstanceNameLabelKey is the key used on pool dependent resources
	CStorPoolInstanceNameLabelKey = "cstorpoolinstance.openebs.io/name"

	// CStorPoolInstanceUIDLabelKey is the key used on pool dependent resources
	CStorPoolInstanceUIDLabelKey = "cstorpoolinstance.openebs.io/uid"

	// PersistentVolumeLabelKey label key set in all cstorvolume replicas of a
	// given volume
	PersistentVolumeLabelKey = "openebs.io/persistent-volume"

	// BlockDeviceTagLabelKey is the key to fetch tag of a block
	// device.
	// For more info : https://github.com/openebs/node-disk-manager/pull/400
	BlockDeviceTagLabelKey = "openebs.io/block-device-tag"
)

const (
	// CSPCFinalizer represents finalizer value used by cspc
	CSPCFinalizer = "cstorpoolcluster.openebs.io/finalizer"

	// PoolProtectionFinalizer is used to make sure cspi and it's bdcs
	// are not deleted before destroying the zpool
	PoolProtectionFinalizer = "openebs.io/pool-protection"

	// CstorVolumeKind is a K8s CR of kind CStorVolume
	CStorVolumeKind = "CStorVolume"

	// CstorVolumeReplicaKind is a K8s CR of kind CStorVolumeReplica
	CStorVolumeReplicaKind = "CStorVolumeReplica"
)

const (
	// CasTypeCStor is the key for cas type cStor
	CasTypeCStor = "cstor"

	// CasTypeJiva is the key for cas type jiva
	CasTypeJiva = "jiva"
)

const (
	CStorPoolBasePath = "/var/openebs/cstor-pool/"
	CacheFileName     = "pool.cache"
)

var (
	// ConfFileMutex is to hold the lock while updating istgt.conf file
	ConfFileMutex = &sync.Mutex{}
	// IstgtConfPath will locate path for istgt configurations
	IstgtConfPath = "/usr/local/etc/istgt/istgt.conf"
	//DesiredReplicationFactorKey is plain text in istgt configuration file informs
	//about desired replication factor used by target
	DesiredReplicationFactorKey = "  DesiredReplicationFactor"
	//TargetNamespace is namespace where target pod and cstorvolume is present
	//and this is updated by addEventHandler function
	TargetNamespace = ""
)

const (
	//IoWaitTime is the time interval for which the IO has to be stopped before doing snapshot operation
	IoWaitTime = 10
	//TotalWaitTime is the max time duration to wait for doing snapshot operation on all the replicas
	TotalWaitTime = 60
)

const (
	// OpenEBSDisableDependantsReconcileKey is the annotation key that decides to create
	// children objects with OpenEBSDisableReconcileKey as true or false
	OpenEBSDisableDependantsReconcileKey = "reconcile.openebs.io/disable-dependants"

	// OpenEBSCStorExistingPoolName is the name of the cstor pool already present on
	// the disk that needs to be imported and renamed
	OpenEBSCStorExistingPoolName = "import.cspi.cstor.openebs.io/existing-pool-name"

	// OpenEBSCStorAllowedBDTagKey is the annotation key present that decides whether
	// a particular BD with a tag is allowed in storage provisioning or not.
	// This annotation can be used on SPC or CSPC to allow a particular BD(s) with tag
	// for provisioning.
	OpenEBSAllowedBDTagKey = "openebs.io/allowed-bd-tags"
)
