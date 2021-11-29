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
	"github.com/openebs/api/v3/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StoragePoolKindCSPC = "CStorPoolCluster"
	// APIVersion holds the value of OpenEBS version
	APIVersion = "cstor.openebs.io/v1"
)

func NewCStorPoolInstance() *CStorPoolInstance {
	return &CStorPoolInstance{}
}

// WithName sets the Name field of CSPI with provided value.
func (cspi *CStorPoolInstance) WithName(name string) *CStorPoolInstance {
	cspi.Name = name
	return cspi
}

// WithNamespace sets the Namespace field of CSPI provided arguments
func (cspi *CStorPoolInstance) WithNamespace(namespace string) *CStorPoolInstance {
	cspi.Namespace = namespace
	return cspi
}

// WithAnnotationsNew sets the Annotations field of CSPI with provided arguments
func (cspi *CStorPoolInstance) WithAnnotationsNew(annotations map[string]string) *CStorPoolInstance {
	cspi.Annotations = make(map[string]string)
	for key, value := range annotations {
		cspi.Annotations[key] = value
	}
	return cspi
}

// WithAnnotations appends or overwrites existing Annotations
// values of CSPI with provided arguments
func (cspi *CStorPoolInstance) WithAnnotations(annotations map[string]string) *CStorPoolInstance {

	if cspi.Annotations == nil {
		return cspi.WithAnnotationsNew(annotations)
	}
	for key, value := range annotations {
		cspi.Annotations[key] = value
	}
	return cspi
}

// WithLabelsNew sets the Labels field of CSPI with provided arguments
func (cspi *CStorPoolInstance) WithLabelsNew(labels map[string]string) *CStorPoolInstance {
	cspi.Labels = make(map[string]string)
	for key, value := range labels {
		cspi.Labels[key] = value
	}
	return cspi
}

// WithLabels appends or overwrites existing Labels
// values of CSPI with provided arguments
func (cspi *CStorPoolInstance) WithLabels(labels map[string]string) *CStorPoolInstance {
	if cspi.Labels == nil {
		return cspi.WithLabelsNew(labels)
	}
	for key, value := range labels {
		cspi.Labels[key] = value
	}
	return cspi
}

// WithNodeSelectorByReference sets the node selector field of CSPI with provided argument.
func (cspi *CStorPoolInstance) WithNodeSelectorByReference(nodeSelector map[string]string) *CStorPoolInstance {
	cspi.Spec.NodeSelector = nodeSelector
	return cspi
}

// WithNodeName sets the HostName field of CSPI with the provided argument.
func (cspi *CStorPoolInstance) WithNodeName(nodeName string) *CStorPoolInstance {
	cspi.Spec.HostName = nodeName
	return cspi
}

// WithPoolConfig sets the pool config field of the CSPI with the provided config.
func (cspi *CStorPoolInstance) WithPoolConfig(poolConfig PoolConfig) *CStorPoolInstance {
	cspi.Spec.PoolConfig = poolConfig
	return cspi
}

// WithDataRaidGroups sets the DataRaidGroups of the CSPI with the provided raid groups.
func (cspi *CStorPoolInstance) WithDataRaidGroups(raidGroup []RaidGroup) *CStorPoolInstance {
	cspi.Spec.DataRaidGroups = raidGroup
	return cspi
}

// WithWriteCacheRaidGroups sets the WriteCacheRaidGroups of the CSPI with the provided raid groups.
func (cspi *CStorPoolInstance) WithWriteCacheRaidGroups(raidGroup []RaidGroup) *CStorPoolInstance {
	cspi.Spec.WriteCacheRaidGroups = raidGroup
	return cspi
}

// WithFinalizer sets the finalizer field in the BDC
func (cspi *CStorPoolInstance) WithFinalizer(finalizers ...string) *CStorPoolInstance {
	cspi.Finalizers = append(cspi.Finalizers, finalizers...)
	return cspi
}

// WithCSPCOwnerReference sets the OwnerReference field in CSPI with required
//fields
func (cspi *CStorPoolInstance) WithCSPCOwnerReference(reference metav1.OwnerReference) *CStorPoolInstance {
	cspi.OwnerReferences = append(cspi.OwnerReferences, reference)
	return cspi
}

// WithNewVersion sets the current and desired version field of
// CSPI with provided arguments
func (cspi *CStorPoolInstance) WithNewVersion(version string) *CStorPoolInstance {
	cspi.VersionDetails.Status.Current = version
	cspi.VersionDetails.Desired = version
	return cspi
}

// WithDependentsUpgraded sets the field to true for new CSPI
func (cspi *CStorPoolInstance) WithDependentsUpgraded() *CStorPoolInstance {
	cspi.VersionDetails.Status.DependentsUpgraded = true
	return cspi
}

// HasFinalizer returns true if the provided finalizer is present on the object.
func (cspi *CStorPoolInstance) HasFinalizer(finalizer string) bool {
	finalizersList := cspi.GetFinalizers()
	return util.ContainsString(finalizersList, finalizer)
}

// RemoveFinalizer removes the given finalizer from the object.
func (cspi *CStorPoolInstance) RemoveFinalizer(finalizer string) {
	cspi.Finalizers = util.RemoveString(cspi.Finalizers, finalizer)
}

// IsDestroyed returns true if CSPI is deletion time stamp is set
func (cspi *CStorPoolInstance) IsDestroyed() bool {
	return !cspi.DeletionTimestamp.IsZero()
}

// IsEmptyStatus is to check whether the status of cStorPoolInstance is empty.
func (cspi *CStorPoolInstance) IsEmptyStatus() bool {
	return cspi.Status.Phase == CStorPoolStatusEmpty
}

// IsPendingStatus is to check if the status of cStorPoolInstance is pending.
func (cspi *CStorPoolInstance) IsPendingStatus() bool {
	return cspi.Status.Phase == CStorPoolStatusPending
}

// IsOnlineStatus is to check if the status of cStorPoolInstance is online.
func (cspi *CStorPoolInstance) IsOnlineStatus() bool {
	return cspi.Status.Phase == CStorPoolStatusOnline
}

// GetAllRaidGroups returns list of all raid groups presents in cspi
func (cspi *CStorPoolInstance) GetAllRaidGroups() []RaidGroup {
	var rgs []RaidGroup
	rgs = append(rgs, cspi.Spec.DataRaidGroups...)
	rgs = append(rgs, cspi.Spec.WriteCacheRaidGroups...)
	return rgs
}

// HasAnnotation return true if provided annotation
// key and value are present on the object.
func (cspi *CStorPoolInstance) HasAnnotation(key, value string) bool {
	val, ok := cspi.GetAnnotations()[key]
	if ok {
		return val == value
	}
	return false
}

// HasLabel returns true if provided label
// key and value are present on the object.
func (cspi *CStorPoolInstance) HasLabel(key, value string) bool {
	val, ok := cspi.GetLabels()[key]
	if ok {
		return val == value
	}
	return false
}

// HasNodeName returns true if the CSPI belongs
// to the provided node name.
func (cspi *CStorPoolInstance) HasNodeName(nodeName string) bool {
	return cspi.Spec.HostName == nodeName
}

// HasNodeName is predicate to filter out based on
// node name of CSPI instances.
func HasNodeName(nodeName string) CSPIPredicate {
	return func(cspi *CStorPoolInstance) bool {
		return cspi.HasNodeName(nodeName)
	}
}

// IsOnline is predicate to filter out based on
// online CSPI instances.
func IsOnline() CSPIPredicate {
	return func(cspi *CStorPoolInstance) bool {
		return cspi.IsOnlineStatus()
	}
}

// Predicate defines an abstraction to determine conditional checks against the
// provided CStorPoolInstance
type CSPIPredicate func(*CStorPoolInstance) bool

// PredicateList holds the list of Predicates
type cspiPredicateList []CSPIPredicate

// all returns true if all the predicates succeed against the provided block
// device instance.
func (l cspiPredicateList) all(cspi *CStorPoolInstance) bool {
	for _, pred := range l {
		if !pred(cspi) {
			return false
		}
	}
	return true
}

// Filter will filter the csp instances
// if all the predicates succeed against that
// csp.
func (cspiList *CStorPoolInstanceList) Filter(p ...CSPIPredicate) *CStorPoolInstanceList {
	var plist cspiPredicateList
	plist = append(plist, p...)
	if len(plist) == 0 {
		return cspiList
	}

	filtered := &CStorPoolInstanceList{}
	for _, cspi := range cspiList.Items {
		cspi := cspi // pin it
		if plist.all(&cspi) {
			filtered.Items = append(filtered.Items, cspi)
		}
	}
	return filtered
}
