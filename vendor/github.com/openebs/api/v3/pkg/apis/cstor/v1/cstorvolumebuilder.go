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
	"fmt"
	"strings"

	"github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/api/v3/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (

	//CStorNodeBase nodeBase for cstor volume
	CStorNodeBase string = "iqn.2016-09.com.openebs.cstor"

	// TargetPort is port for cstor volume
	TargetPort string = "3260"

	// CStorVolumeReplicaFinalizer is the name of finalizer on CStorVolumeConfig
	CStorVolumeReplicaFinalizer = "cstorvolumereplica.openebs.io/finalizer"
)

// NewCStorVolumeConfig returns new instance of CStorVolumeConfig
func NewCStorVolumeConfig() *CStorVolumeConfig {
	return &CStorVolumeConfig{}
}

// WithName sets the Name field of CVC with provided value.
func (cvc *CStorVolumeConfig) WithName(name string) *CStorVolumeConfig {
	cvc.Name = name
	return cvc
}

// WithNamespace sets the Namespace field of CVC provided arguments
func (cvc *CStorVolumeConfig) WithNamespace(namespace string) *CStorVolumeConfig {
	cvc.Namespace = namespace
	return cvc
}

// WithAnnotationsNew sets the Annotations field of CVC with provided arguments
func (cvc *CStorVolumeConfig) WithAnnotationsNew(annotations map[string]string) *CStorVolumeConfig {
	cvc.Annotations = make(map[string]string)
	for key, value := range annotations {
		cvc.Annotations[key] = value
	}
	return cvc
}

// WithAnnotations appends or overwrites existing Annotations
// values of CVC with provided arguments
func (cvc *CStorVolumeConfig) WithAnnotations(annotations map[string]string) *CStorVolumeConfig {

	if cvc.Annotations == nil {
		return cvc.WithAnnotationsNew(annotations)
	}
	for key, value := range annotations {
		cvc.Annotations[key] = value
	}
	return cvc
}

// WithLabelsNew sets the Lacvels field of CVC with provided arguments
func (cvc *CStorVolumeConfig) WithLabelsNew(labels map[string]string) *CStorVolumeConfig {
	cvc.Labels = make(map[string]string)
	for key, value := range labels {
		cvc.Labels[key] = value
	}
	return cvc
}

// WithLabels appends or overwrites existing Lacvels
// values of CVC with provided arguments
func (cvc *CStorVolumeConfig) WithLabels(labels map[string]string) *CStorVolumeConfig {
	if cvc.Labels == nil {
		return cvc.WithLabelsNew(labels)
	}
	for key, value := range labels {
		cvc.Labels[key] = value
	}
	return cvc
}

// WithFinalizer sets the finalizer field in the CVC
func (cvc *CStorVolumeConfig) WithFinalizer(finalizers ...string) *CStorVolumeConfig {
	cvc.Finalizers = append(cvc.Finalizers, finalizers...)
	return cvc
}

// WithOwnerReference sets the OwnerReference field in CVC with required
//fields
func (cvc *CStorVolumeConfig) WithOwnerReference(reference metav1.OwnerReference) *CStorVolumeConfig {
	cvc.OwnerReferences = append(cvc.OwnerReferences, reference)
	return cvc
}

// WithNewVersion sets the current and desired version field of
// CVC with provided arguments
func (cvc *CStorVolumeConfig) WithNewVersion(version string) *CStorVolumeConfig {
	cvc.VersionDetails.Status.Current = version
	cvc.VersionDetails.Desired = version
	return cvc
}

// WithDependentsUpgraded sets the field to true for new CVC
func (cvc *CStorVolumeConfig) WithDependentsUpgraded() *CStorVolumeConfig {
	cvc.VersionDetails.Status.DependentsUpgraded = true
	return cvc
}

// HasFinalizer returns true if the provided finalizer is present on the ocvject.
func (cvc *CStorVolumeConfig) HasFinalizer(finalizer string) bool {
	finalizersList := cvc.GetFinalizers()
	return util.ContainsString(finalizersList, finalizer)
}

// RemoveFinalizer removes the given finalizer from the ocvject.
func (cvc *CStorVolumeConfig) RemoveFinalizer(finalizer string) {
	cvc.Finalizers = util.RemoveString(cvc.Finalizers, finalizer)
}

// GetDesiredReplicaPoolNames returns list of desired pool names
func (cvc *CStorVolumeConfig) GetDesiredReplicaPoolNames() []string {
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

// NewCStorVolume returns new instance of CStorVolume
func NewCStorVolume() *CStorVolume {
	return &CStorVolume{}
}

// NewCStorVolumeObj returns a new instance of cstorvolume
func NewCStorVolumeObj(obj *CStorVolume) *CStorVolume {
	return obj
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

// WithLabelsNew sets the Lacvels field of CV with provided arguments
func (cv *CStorVolume) WithLabelsNew(labels map[string]string) *CStorVolume {
	cv.Labels = make(map[string]string)
	for key, value := range labels {
		cv.Labels[key] = value
	}
	return cv
}

// WithLabels appends or overwrites existing Lacvels
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

// WithFinalizers sets the finalizer field in the CV
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

// IsResizePending return true if resize is in progress
func (cv *CStorVolume) IsResizePending() bool {
	curCapacity := cv.Status.Capacity
	desiredCapacity := cv.Spec.Capacity
	// Cmp returns 0 if the curCapacity is equal to desiredCapacity,
	// -1 if the curCapacity is less than desiredCapacity, or 1 if the
	// curCapacity is greater than desiredCapacity.
	return curCapacity.Cmp(desiredCapacity) == -1
}

// IsDRFPending return true if drf update is required else false
// Steps to verify whether drf is required
// 1. Read DesiredReplicationFactor configurations from istgt conf file
// 2. Compare the value with spec.DesiredReplicationFactor and return result
func (cv *CStorVolume) IsDRFPending() bool {
	fileOperator := util.RealFileOperator{}
	types.ConfFileMutex.Lock()
	//If it has proper config then we will get --> "  DesiredReplicationFactor 3"
	i, gotConfig, err := fileOperator.GetLineDetails(types.IstgtConfPath, types.DesiredReplicationFactorKey)
	types.ConfFileMutex.Unlock()
	if err != nil || i == -1 {
		klog.Infof("failed to get %s config details error: %v",
			types.DesiredReplicationFactorKey,
			err,
		)
		return false
	}
	drfStringValue := fmt.Sprintf(" %d", cv.Spec.DesiredReplicationFactor)
	// gotConfig will have "  DesiredReplicationFactor  3" and we will extract
	// numeric character from output
	if !strings.HasSuffix(gotConfig, drfStringValue) {
		return true
	}
	// reconciliation check for replica scaledown scenarion
	return (len(cv.Spec.ReplicaDetails.KnownReplicas) <
		len(cv.Status.ReplicaDetails.KnownReplicas))
}

// GetCVCondition returns corresponding cstorvolume condition based argument passed
func (cv *CStorVolume) GetCVCondition(
	condType CStorVolumeConditionType) CStorVolumeCondition {
	for _, cond := range cv.Status.Conditions {
		if condType == cond.Type {
			return cond
		}
	}
	return CStorVolumeCondition{}
}

// IsConditionPresent returns true if condition is available
func (cv *CStorVolume) IsConditionPresent(condType CStorVolumeConditionType) bool {
	for _, cond := range cv.Status.Conditions {
		if condType == cond.Type {
			return true
		}
	}
	return false
}

// AreSpecReplicasHealthy return true if all the spec replicas are in Healthy
// state else return false
func (cv *CStorVolume) AreSpecReplicasHealthy(volStatus *CVStatus) bool {
	var isReplicaExist bool
	var replicaInfo ReplicaStatus
	for _, replicaValue := range cv.Spec.ReplicaDetails.KnownReplicas {
		isReplicaExist = false
		for _, replicaInfo = range volStatus.ReplicaStatuses {
			if replicaInfo.ID == replicaValue {
				isReplicaExist = true
				break
			}
		}
		if (isReplicaExist && replicaInfo.Mode != "Healthy") || !isReplicaExist {
			return false
		}
	}
	return true
}

// GetRemovingReplicaID return replicaID that present in status but not in spec
func (cv *CStorVolume) GetRemovingReplicaID() string {
	for repID := range cv.Status.ReplicaDetails.KnownReplicas {
		// If known replica is not exist in spec but if it exist in status then
		// user/operator selected that replica for removal
		if _, isReplicaExist :=
			cv.Spec.ReplicaDetails.KnownReplicas[repID]; !isReplicaExist {
			return string(repID)
		}
	}
	return ""
}

// BuildScaleDownConfigData build data based on replica that needs to remove
func (cv *CStorVolume) BuildScaleDownConfigData(repID string) map[string]string {
	configData := map[string]string{}
	newReplicationFactor := cv.Spec.DesiredReplicationFactor
	newConsistencyFactor := (newReplicationFactor / 2) + 1
	key := fmt.Sprintf("  ReplicationFactor")
	value := fmt.Sprintf("  ReplicationFactor %d", newReplicationFactor)
	configData[key] = value
	key = fmt.Sprintf("  ConsistencyFactor")
	value = fmt.Sprintf("  ConsistencyFactor %d", newConsistencyFactor)
	configData[key] = value
	key = fmt.Sprintf("  DesiredReplicationFactor")
	value = fmt.Sprintf("  DesiredReplicationFactor %d",
		cv.Spec.DesiredReplicationFactor)
	configData[key] = value
	key = fmt.Sprintf("  Replica %s", repID)
	value = fmt.Sprintf("")
	configData[key] = value
	return configData
}

// Conditions enables building CRUD operations on cstorvolume conditions
type Conditions []CStorVolumeCondition

// AddCondition appends the new condition to existing conditions
func (c Conditions) AddCondition(cond CStorVolumeCondition) []CStorVolumeCondition {
	c = append(c, cond)
	return c
}

// UpdateCondition updates the condition if it is present in Conditions
func (c Conditions) UpdateCondition(cond CStorVolumeCondition) []CStorVolumeCondition {
	for i, condObj := range c {
		if condObj.Type == cond.Type {
			c[i] = cond
		}
	}
	return c
}

// DeleteCondition deletes the condition from conditions
func (c Conditions) DeleteCondition(cond CStorVolumeCondition) []CStorVolumeCondition {
	newConditions := []CStorVolumeCondition{}
	for _, condObj := range c {
		if condObj.Type != cond.Type {
			newConditions = append(newConditions, condObj)
		}
	}
	return newConditions
}

// GetResizeCondition will return resize condtion related to
// cstorvolume condtions
func GetResizeCondition() CStorVolumeCondition {
	resizeConditions := CStorVolumeCondition{
		Type:               CStorVolumeResizing,
		Status:             ConditionInProgress,
		LastProbeTime:      metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             "Resizing",
		Message:            "Triggered resize by changing capacity in spec",
	}
	return resizeConditions
}

// ****************************************************************************
//
//                       Version Details
//
//
// ****************************************************************************

// SetErrorStatus sets the message and reason for the error
func (vs *VersionStatus) SetErrorStatus(msg string, err error) {
	vs.Message = msg
	vs.Reason = err.Error()
	vs.LastUpdateTime = metav1.Now()
}

// SetInProgressStatus sets the state as ReconcileInProgress
func (vs *VersionStatus) SetInProgressStatus() {
	vs.State = ReconcileInProgress
	vs.LastUpdateTime = metav1.Now()
}

// SetSuccessStatus resets the message and reason and sets the state as
// Reconciled
func (vd *VersionDetails) SetSuccessStatus() {
	vd.Status.Current = vd.Desired
	vd.Status.Message = ""
	vd.Status.Reason = ""
	vd.Status.State = ReconcileComplete
	vd.Status.LastUpdateTime = metav1.Now()
}

// **************************************************************************
//
//                                CSTOR VOLUMES REPLICA
//
//
// **************************************************************************

// CVRPredicate defines an abstraction to determine conditional checks against the
// provided CVolumeReplicas
type CVRPredicate func(*CStorVolumeReplica) bool

// +k8s:deepcopy-gen=false

// CVRPredicateList holds the list of Predicates
type CVRPredicateList []CVRPredicate

// all returns true if all the predicates succeed against the provided list.
func (l CVRPredicateList) all(cvr *CStorVolumeReplica) bool {
	for _, pred := range l {
		if !pred(cvr) {
			return false
		}
	}
	return true
}

// Filter will filter the CVRs if all the predicates succeed
// against that CVRs
func (cvrList *CStorVolumeReplicaList) Filter(p ...CVRPredicate) *CStorVolumeReplicaList {
	var plist CVRPredicateList
	plist = append(plist, p...)
	if len(plist) == 0 {
		return cvrList
	}

	filtered := &CStorVolumeReplicaList{}
	for _, cvr := range cvrList.Items {
		cvr := cvr // pin it
		if plist.all(&cvr) {
			filtered.Items = append(filtered.Items, cvr)
		}
	}
	return filtered
}

// NewCStorVolumeReplica returns new instance of CStorVolumeReplica
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

// WithLabelsNew sets the Lacvrels field of CV with provided arguments
func (cvr *CStorVolumeReplica) WithLabelsNew(labels map[string]string) *CStorVolumeReplica {
	cvr.Labels = make(map[string]string)
	for key, value := range labels {
		cvr.Labels[key] = value
	}
	return cvr
}

// WithLabels appends or overwrites existing Lacvrels
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

// WithFinalizers sets the finalizer field in the CV
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

// WithZvolWorkers sets the zvolworkers with the provided arguments
func (cvr *CStorVolumeReplica) WithZvolWorkers(zvolworker string) *CStorVolumeReplica {
	cvr.Spec.ZvolWorkers = zvolworker
	return cvr
}

// WithCWithCompression sets the compression algorithm with the provided arguments
func (cvr *CStorVolumeReplica) WithCompression(compression string) *CStorVolumeReplica {
	cvr.Spec.Compression = compression
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
func (cvrList *CStorVolumeReplicaList) GetPoolNames() []string {
	poolNames := []string{}
	for _, cvrObj := range cvrList.Items {
		poolNames = append(poolNames, cvrObj.Labels[string(types.CStorPoolInstanceNameLabelKey)])
	}
	return poolNames
}

// IsCVRHealthy returns true if CVR phase is Healthy
func IsCVRHealthy(cvr *CStorVolumeReplica) bool {
	return cvr.Status.Phase == CVRStatusOnline
}

// IsReplicaNonQuorum returns true if CVR phase is
// either NewReplicaDegraded or ReconstructingNewReplica
func IsReplicaNonQuorum(cvr *CStorVolumeReplica) bool {
	if cvr.Status.Phase == CVRStatusNewReplicaDegraded {
		return true
	} else if cvr.Status.Phase == CVRStatusReconstructingNewReplica {
		return true
	}
	return false
}
