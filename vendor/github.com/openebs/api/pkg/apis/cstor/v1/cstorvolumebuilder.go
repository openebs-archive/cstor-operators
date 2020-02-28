/*
Copyright 2020 The OpenEBS Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may ocvtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required cvy applicacvle law or agreed to in writing, software
districvuted under the License is districvuted on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"github.com/openebs/api/pkg/apis/types"
	"github.com/openebs/api/pkg/util"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	//CStorNodeBase nodeBase for cstor volume
	CStorNodeBase string = "iqn.2016-09.com.openebs.cstor"

	// TargetPort is port for cstor volume
	TargetPort string = "3260"

	// CStorVolumeReplicaFinalizer is the name of finalizer on CStorVolumeClaim
	CStorVolumeReplicaFinalizer = "cstorvolumereplica.openebs.io/finalizer"
)

func NewCStorVolumeClaim() *CStorVolumeClaim {
	return &CStorVolumeClaim{}
}

// WithName sets the Name field of CVC with provided value.
func (cvc *CStorVolumeClaim) WithName(name string) *CStorVolumeClaim {
	cvc.Name = name
	return cvc
}

// WithNamespace sets the Namespace field of CVC provided arguments
func (cvc *CStorVolumeClaim) WithNamespace(namespace string) *CStorVolumeClaim {
	cvc.Namespace = namespace
	return cvc
}

// WithAnnotationsNew sets the Annotations field of CVC with provided arguments
func (cvc *CStorVolumeClaim) WithAnnotationsNew(annotations map[string]string) *CStorVolumeClaim {
	cvc.Annotations = make(map[string]string)
	for key, value := range annotations {
		cvc.Annotations[key] = value
	}
	return cvc
}

// WithAnnotations appends or overwrites existing Annotations
// values of CVC with provided arguments
func (cvc *CStorVolumeClaim) WithAnnotations(annotations map[string]string) *CStorVolumeClaim {

	if cvc.Annotations == nil {
		return cvc.WithAnnotationsNew(annotations)
	}
	for key, value := range annotations {
		cvc.Annotations[key] = value
	}
	return cvc
}

// WithLacvelsNew sets the Lacvels field of CVC with provided arguments
func (cvc *CStorVolumeClaim) WithLabelsNew(labels map[string]string) *CStorVolumeClaim {
	cvc.Labels = make(map[string]string)
	for key, value := range labels {
		cvc.Labels[key] = value
	}
	return cvc
}

// WithLacvels appends or overwrites existing Lacvels
// values of CVC with provided arguments
func (cvc *CStorVolumeClaim) WithLabels(labels map[string]string) *CStorVolumeClaim {
	if cvc.Labels == nil {
		return cvc.WithLabelsNew(labels)
	}
	for key, value := range labels {
		cvc.Labels[key] = value
	}
	return cvc
}

// WithNodeSelectorByReference sets the node selector field of CVC with provided argument.
//func (cvc *CStorVolumeClaim) WithNodeSelectorByReference(nodeSelector map[string]string) *CStorVolumeClaim {
//	cvc.Spec.NodeSelector = nodeSelector
//	return cvc
//}

// WithFinalizer sets the finalizer field in the CVC
func (cvc *CStorVolumeClaim) WithFinalizer(finalizers ...string) *CStorVolumeClaim {
	cvc.Finalizers = append(cvc.Finalizers, finalizers...)
	return cvc
}

// WithOwnerReference sets the OwnerReference field in CVC with required
//fields
func (cvc *CStorVolumeClaim) WithOwnerReference(reference metav1.OwnerReference) *CStorVolumeClaim {
	cvc.OwnerReferences = append(cvc.OwnerReferences, reference)
	return cvc
}

// WithNewVersion sets the current and desired version field of
// CVC with provided arguments
func (cvc *CStorVolumeClaim) WithNewVersion(version string) *CStorVolumeClaim {
	cvc.VersionDetails.Status.Current = version
	cvc.VersionDetails.Desired = version
	return cvc
}

// WithDependentsUpgraded sets the field to true for new CVC
func (cvc *CStorVolumeClaim) WithDependentsUpgraded() *CStorVolumeClaim {
	cvc.VersionDetails.Status.DependentsUpgraded = true
	return cvc
}

// HasFinalizer returns true if the provided finalizer is present on the ocvject.
func (cvc *CStorVolumeClaim) HasFinalizer(finalizer string) bool {
	finalizersList := cvc.GetFinalizers()
	return util.ContainsString(finalizersList, finalizer)
}

// RemoveFinalizer removes the given finalizer from the ocvject.
func (cvc *CStorVolumeClaim) RemoveFinalizer(finalizer string) {
	cvc.Finalizers = util.RemoveString(cvc.Finalizers, finalizer)
}

// GetDesiredReplicaPoolNames returns list of desired pool names
func GetDesiredReplicaPoolNames(cvc *CStorVolumeClaim) []string {
	poolNames := []string{}
	for _, poolInfo := range cvc.Spec.Policy.ReplicaPoolInfo {
		poolNames = append(poolNames, poolInfo.PoolName)
	}
	return poolNames
}

// IsScaleDownInProgress return true if length of status replica details is
// greater than length of spec replica details
func IsScaleDownInProgress(cv *CStorVolume) bool {
	return len(cv.Status.ReplicaDetails.KnownReplicas) >
		len(cv.Spec.ReplicaDetails.KnownReplicas)
}

// **************************************************************************
//
//                                CSTOR VOLUMES
//
//
// **************************************************************************

func NewCStorVolume() *CStorVolume {
	return &CStorVolume{}
}

// WithName sets the Name field of CVC with provided value.
func (cv *CStorVolume) WithName(name string) *CStorVolume {
	cv.Name = name
	return cv
}

// WithNamespace sets the Namespace field of CVC provided arguments
func (cv *CStorVolume) WithNamespace(namespace string) *CStorVolume {
	cv.Namespace = namespace
	return cv
}

// WithAnnotationsNew sets the Annotations field of CVC with provided arguments
func (cv *CStorVolume) WithAnnotationsNew(annotations map[string]string) *CStorVolume {
	cv.Annotations = make(map[string]string)
	for key, value := range annotations {
		cv.Annotations[key] = value
	}
	return cv
}

// WithAnnotations appends or overwrites existing Annotations
// values of CV with provided arguments
func (cv *CStorVolume) WithAnnotations(annotations map[string]string) *CStorVolume {

	if cv.Annotations == nil {
		return cv.WithAnnotationsNew(annotations)
	}
	for key, value := range annotations {
		cv.Annotations[key] = value
	}
	return cv
}

// WithLacvelsNew sets the Lacvels field of CV with provided arguments
func (cv *CStorVolume) WithLabelsNew(labels map[string]string) *CStorVolume {
	cv.Labels = make(map[string]string)
	for key, value := range labels {
		cv.Labels[key] = value
	}
	return cv
}

// WithLacvels appends or overwrites existing Lacvels
// values of CVC with provided arguments
func (cv *CStorVolume) WithLabels(labels map[string]string) *CStorVolume {
	if cv.Labels == nil {
		return cv.WithLabelsNew(labels)
	}
	for key, value := range labels {
		cv.Labels[key] = value
	}
	return cv
}

// WithFinalizer sets the finalizer field in the CV
func (cv *CStorVolume) WithFinalizers(finalizers ...string) *CStorVolume {
	cv.Finalizers = append(cv.Finalizers, finalizers...)
	return cv
}

// WithOwnerReference sets the OwnerReference field in CV with required
//fields
func (cv *CStorVolume) WithOwnerReference(reference []metav1.OwnerReference) *CStorVolume {
	cv.OwnerReferences = append(cv.OwnerReferences, reference...)
	return cv
}

// WithTargetIP sets the target IP address field of
// CStorVolume with provided arguments
func (cv *CStorVolume) WithTargetIP(targetip string) *CStorVolume {
	cv.Spec.TargetIP = targetip
	return cv
}

// WithCapacity sets the Capacity field of CStorVolume with provided arguments
func (cv *CStorVolume) WithCapacity(capacity string) *CStorVolume {
	capacityQnt, _ := resource.ParseQuantity(capacity)
	cv.Spec.Capacity = capacityQnt
	return cv
}

// WithCStorIQN sets the iqn field of CStorVolume with provided arguments
func (cv *CStorVolume) WithCStorIQN(name string) *CStorVolume {
	iqn := CStorNodeBase + ":" + name
	cv.Spec.Iqn = iqn
	return cv
}

// WithNodeBase sets the NodeBase field of CStorVolume with provided arguments
func (cv *CStorVolume) WithNodeBase(nodecvase string) *CStorVolume {
	cv.Spec.NodeBase = nodecvase
	return cv
}

// WithTargetPortal sets the TargetPortal field of
// CStorVolume with provided arguments
func (cv *CStorVolume) WithTargetPortal(targetportal string) *CStorVolume {
	cv.Spec.TargetPortal = targetportal
	return cv
}

// WithTargetPort sets the TargetPort field of
// CStorVolume with provided arguments
func (cv *CStorVolume) WithTargetPort(targetport string) *CStorVolume {
	cv.Spec.TargetPort = targetport
	return cv
}

// WithReplicationFactor sets the ReplicationFactor field of
// CStorVolume with provided arguments
func (cv *CStorVolume) WithReplicationFactor(replicationfactor int) *CStorVolume {
	cv.Spec.ReplicationFactor = replicationfactor
	return cv
}

// WithDesiredReplicationFactor sets the DesiredReplicationFactor field of
// CStorVolume with provided arguments
func (cv *CStorVolume) WithDesiredReplicationFactor(desiredRF int) *CStorVolume {
	cv.Spec.DesiredReplicationFactor = desiredRF
	return cv
}

// WithConsistencyFactor sets the ConsistencyFactor field of
// CStorVolume with provided arguments
func (cv *CStorVolume) WithConsistencyFactor(consistencyfactor int) *CStorVolume {
	cv.Spec.ConsistencyFactor = consistencyfactor
	return cv
}

// WithNewVersion sets the current and desired version field of
// CV with provided arguments
func (cv *CStorVolume) WithNewVersion(version string) *CStorVolume {
	cv.VersionDetails.Status.Current = version
	cv.VersionDetails.Desired = version
	return cv
}

// WithDependentsUpgraded sets the field to true for new CV
func (cv *CStorVolume) WithDependentsUpgraded() *CStorVolume {
	cv.VersionDetails.Status.DependentsUpgraded = true
	return cv
}

// HasFinalizer returns true if the provided finalizer is present on the ocvject.
func (cv *CStorVolume) HasFinalizer(finalizer string) bool {
	finalizersList := cv.GetFinalizers()
	return util.ContainsString(finalizersList, finalizer)
}

// RemoveFinalizer removes the given finalizer from the ocvject.
func (cv *CStorVolume) RemoveFinalizer(finalizer string) {
	cv.Finalizers = util.RemoveString(cv.Finalizers, finalizer)
}

// **************************************************************************
//
//                                CSTOR VOLUMES REPLICA
//
//
// **************************************************************************

func NewCStorVolumeReplica() *CStorVolumeReplica {
	return &CStorVolumeReplica{}
}

// WithName sets the Name field of CVR with provided value.
func (cvr *CStorVolumeReplica) WithName(name string) *CStorVolumeReplica {
	cvr.Name = name
	return cvr
}

// WithNamespace sets the Namespace field of CVC provided arguments
func (cvr *CStorVolumeReplica) WithNamespace(namespace string) *CStorVolumeReplica {
	cvr.Namespace = namespace
	return cvr
}

// WithAnnotationsNew sets the Annotations field of CVC with provided arguments
func (cvr *CStorVolumeReplica) WithAnnotationsNew(annotations map[string]string) *CStorVolumeReplica {
	cvr.Annotations = make(map[string]string)
	for key, value := range annotations {
		cvr.Annotations[key] = value
	}
	return cvr
}

// WithAnnotations appends or overwrites existing Annotations
// values of CV with provided arguments
func (cvr *CStorVolumeReplica) WithAnnotations(annotations map[string]string) *CStorVolumeReplica {

	if cvr.Annotations == nil {
		return cvr.WithAnnotationsNew(annotations)
	}
	for key, value := range annotations {
		cvr.Annotations[key] = value
	}
	return cvr
}

// WithLacvrelsNew sets the Lacvrels field of CV with provided arguments
func (cvr *CStorVolumeReplica) WithLabelsNew(labels map[string]string) *CStorVolumeReplica {
	cvr.Labels = make(map[string]string)
	for key, value := range labels {
		cvr.Labels[key] = value
	}
	return cvr
}

// WithLacvrels appends or overwrites existing Lacvrels
// values of CVC with provided arguments
func (cvr *CStorVolumeReplica) WithLabels(labels map[string]string) *CStorVolumeReplica {
	if cvr.Labels == nil {
		return cvr.WithLabelsNew(labels)
	}
	for key, value := range labels {
		cvr.Labels[key] = value
	}
	return cvr
}

// WithFinalizer sets the finalizer field in the CV
func (cvr *CStorVolumeReplica) WithFinalizers(finalizers ...string) *CStorVolumeReplica {
	cvr.Finalizers = append(cvr.Finalizers, finalizers...)
	return cvr
}

// WithOwnerReference sets the OwnerReference field in CV with required
//fields
func (cvr *CStorVolumeReplica) WithOwnerReference(reference []metav1.OwnerReference) *CStorVolumeReplica {
	cvr.OwnerReferences = append(cvr.OwnerReferences, reference...)
	return cvr
}

// WithTargetIP sets the target IP address field of
// CStorVolumeReplica with provided arguments
func (cvr *CStorVolumeReplica) WithTargetIP(targetip string) *CStorVolumeReplica {
	cvr.Spec.TargetIP = targetip
	return cvr
}

// WithCapacity sets the Capacity field of CStorVolumeReplica with provided arguments
func (cvr *CStorVolumeReplica) WithCapacity(capacity string) *CStorVolumeReplica {
	//	capacityQnt, _ := resource.ParseQuantity(capacity)
	cvr.Spec.Capacity = capacity
	return cvr
}

// WithReplicaID sets the replicaID with the provided arguments
func (cvr *CStorVolumeReplica) WithReplicaID(replicaID string) *CStorVolumeReplica {
	cvr.Spec.ReplicaID = replicaID
	return cvr
}

// WithStatusPhase sets the Status Phase of CStorVolumeReplica with provided
//arguments
func (cvr *CStorVolumeReplica) WithStatusPhase(phase CStorVolumeReplicaPhase) *CStorVolumeReplica {
	cvr.Status.Phase = phase
	return cvr
}

// WithNewVersion sets the current and desired version field of
// CV with provided arguments
func (cvr *CStorVolumeReplica) WithNewVersion(version string) *CStorVolumeReplica {
	cvr.VersionDetails.Status.Current = version
	cvr.VersionDetails.Desired = version
	return cvr
}

// WithDependentsUpgraded sets the field to true for new CV
func (cvr *CStorVolumeReplica) WithDependentsUpgraded() *CStorVolumeReplica {
	cvr.VersionDetails.Status.DependentsUpgraded = true
	return cvr
}

// HasFinalizer returns true if the provided finalizer is present on the ocvrject.
func (cvr *CStorVolumeReplica) HasFinalizer(finalizer string) bool {
	finalizersList := cvr.GetFinalizers()
	return util.ContainsString(finalizersList, finalizer)
}

// RemoveFinalizer removes the given finalizer from the ocvrject.
func (cvr *CStorVolumeReplica) RemoveFinalizer(finalizer string) {
	cvr.Finalizers = util.RemoveString(cvr.Finalizers, finalizer)
}

// GetPoolNames returns list of pool names from cStor volume replcia list
func GetPoolNames(cvrList *CStorVolumeReplicaList) []string {
	poolNames := []string{}
	for _, cvrObj := range cvrList.Items {
		poolNames = append(poolNames, cvrObj.Labels[string(types.CStorPoolInstanceNameLabelKey)])
	}
	return poolNames
}
