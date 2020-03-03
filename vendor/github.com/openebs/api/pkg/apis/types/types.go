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

const (
	// OpenEBSDisableReconcileLabelKey is the label key decides to reconcile or not
	OpenEBSDisableReconcileLabelKey = "reconcile.openebs.io/disable"

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

	// PersistentVolumeLabelKey label key set in all cstorvolume replicas of a
	// given volume
	PersistentVolumeLabelKey = "openebs.io/persistent-volume"
)

const (
	// CSPCFinalizer represents finalizer value used by cspc
	CSPCFinalizer = "cstorpoolcluster.openebs.io/finalizer"

	// PoolProtectionFinalizer is used to make sure cspi and it's bdcs
	// are not deleted before destroying the zpool
	PoolProtectionFinalizer = "openebs.io/pool-protection"
)

const (
	// CasTypeCStor is the key for cas type cStor
	CasTypeCStor = "cstor"

	// CasTypeJiva is the key for cas type jiva
	CasTypeJiva = "jiva"
)

const (
	CStorPoolBasePath = "/var/openebs/cstor-pool"
	CacheFileName     = "pool.cache"
)
