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

package webhook

import (
	"context"
	"encoding/json"
	"net/http"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	util "github.com/openebs/api/v3/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

type validateFunc func(cvcOldObj, cvcNewObj *cstor.CStorVolumeConfig) error

type getCVC func(name, namespace string, clientset clientset.Interface) (*cstor.CStorVolumeConfig, error)

func (wh *webhook) validateCVC(ar *v1.AdmissionReview) *v1.AdmissionResponse {
	req := ar.Request
	response := &v1.AdmissionResponse{}
	response.Allowed = true
	// validates only if requested operation is UPDATE
	if req.Operation == v1.Update {
		return wh.validateCVCUpdateRequest(req, getCVCObject)
	}
	klog.V(4).Info("Admission wehbook for CVC module not " +
		"configured for operations other than UPDATE")
	return response
}

func (wh *webhook) validateCVCUpdateRequest(req *v1.AdmissionRequest, getCVC getCVC) *v1.AdmissionResponse {
	response := NewAdmissionResponse().
		SetAllowed().
		WithResultAsSuccess(http.StatusAccepted).AR
	var cvcNewObj cstor.CStorVolumeConfig
	err := json.Unmarshal(req.Object.Raw, &cvcNewObj)
	if err != nil {
		klog.Errorf("Couldn't unmarshal raw object: %+v to cvc error: %v", string(req.Object.Raw), err)
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusBadRequest).AR
		return response
	}

	// Get old CVC object by making call to etcd
	cvcOldObj, err := getCVC(cvcNewObj.Name, cvcNewObj.Namespace, wh.clientset)
	if err != nil {
		klog.Errorf("Failed to get CVC %s in namespace %s from etcd error: %v", cvcNewObj.Name, cvcNewObj.Namespace, err)
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusBadRequest).AR
		return response
	}
	err = validateCVCSpecChanges(cvcOldObj, &cvcNewObj)
	if err != nil {
		klog.Errorf("invalid cvc changes: %s error: %s", cvcOldObj.Name, err.Error())
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusBadRequest).AR
		return response
	}
	return response
}

func validateCVCSpecChanges(cvcOldObj, cvcNewObj *cstor.CStorVolumeConfig) error {
	validateFuncList := []validateFunc{validateReplicaCount,
		validateProvisionedCapacity,
		validatePoolListChanges,
		validateReplicaScaling,
	}
	for _, f := range validateFuncList {
		err := f(cvcOldObj, cvcNewObj)
		if err != nil {
			return err
		}
	}

	// Below validations should be done only with new CVC object
	return validatePoolNames(cvcNewObj)
}

// TODO: isScalingInProgress(cvcObj *cstor.CStorVolumeConfig) signature need to be
// updated to cvcObj.IsScaleingInProgress()
func isScalingInProgress(cvcObj *cstor.CStorVolumeConfig) bool {
	return len(cvcObj.Spec.Policy.ReplicaPoolInfo) != len(cvcObj.Status.PoolInfo)
}

// validateReplicaCount returns error if user modified the replica count after
// provisioning the volume else return nil
func validateReplicaCount(cvcOldObj, cvcNewObj *cstor.CStorVolumeConfig) error {
	if cvcOldObj.Spec.Provision.ReplicaCount != cvcNewObj.Spec.Provision.ReplicaCount {
		return errors.Errorf(
			"cvc %s replicaCount got modified from %d to %d",
			cvcOldObj.Name,
			cvcOldObj.Spec.Provision.ReplicaCount,
			cvcNewObj.Spec.Provision.ReplicaCount,
		)
	}
	return nil
}

// validateProvisionedCapacity returns error if user modified the initial
// provisioned capacity after provisioning the volume else return nil otherwise
func validateProvisionedCapacity(cvcOldObj, cvcNewObj *cstor.CStorVolumeConfig) error {
	newProvisionedCap := cvcNewObj.Spec.Provision.Capacity[corev1.ResourceStorage]
	oldProvisionedCap := cvcOldObj.Spec.Provision.Capacity[corev1.ResourceStorage]

	if newProvisionedCap.Cmp(oldProvisionedCap) != 0 {
		return errors.Errorf(
			"cvc initial provisioned capacity `Spec.Provision.Capacity` is immutable, can't be modified ")
	}
	return nil
}

// validatePoolListChanges returns error if user modified existing pool names with new
// pool name(s) or if user performed more than one replica scale down at a time
func validatePoolListChanges(cvcOldObj, cvcNewObj *cstor.CStorVolumeConfig) error {
	// Check the new CVC spec changes with old CVC status(Comparing with status
	// is more appropriate than comparing with spec)
	oldCurrentPoolNames := cvcOldObj.Status.PoolInfo
	newDesiredPoolNames := cvcNewObj.GetDesiredReplicaPoolNames()
	modifiedPoolNames := util.ListDiff(oldCurrentPoolNames, newDesiredPoolNames)
	// Reject the request if someone perform scaling when CVC is not in Bound
	// state
	// NOTE: We should not reject the controller request which Updates status as
	// Bound as well as pool info in status and spec
	// TODO: Make below check as cvcOldObj.ISBound()
	// If CVC Status is not bound then reject
	if cvcOldObj.Status.Phase != cstor.CStorVolumeConfigPhaseBound {
		// If controller is updating pool info then new CVC will be in bound state
		if cvcNewObj.Status.Phase != cstor.CStorVolumeConfigPhaseBound &&
			// Performed scaling operation on CVC
			len(oldCurrentPoolNames) != len(newDesiredPoolNames) {
			return errors.Errorf(
				"Can't perform scaling of volume replicas when CVC is not in %s state",
				cstor.CStorVolumeConfigPhaseBound,
			)
		}
	}

	// Validing Scaling process
	if len(newDesiredPoolNames) >= len(oldCurrentPoolNames) {
		// If no.of pools on new spec >= no.of pools in old status(scaleup as well
		// as migration case then all the pools in old status must present in new
		// spec)
		if len(modifiedPoolNames) > 0 {
			return errors.Errorf(
				"volume replica migration directly by modifying pool names %v is not yet supported",
				modifiedPoolNames,
			)
		}
	} else {
		// If no.of pools in new spec < no.of pools in old status(scale down
		// volume replica case) then there should at most one change in
		// oldSpec.PoolInfo - newSpec.PoolInfo
		if len(modifiedPoolNames) > 1 {
			return errors.Errorf(
				"Can't perform more than one replica scale down requested scale down count %d",
				len(modifiedPoolNames),
			)
		}
	}
	return nil
}

// validateReplicaScaling returns error if user updated pool list when scaling is
// already in progress.
// Note: User can perform scaleup of multiple replicas by adding multiple pool
//       names at time but not by updating CVC pool names with multiple edits.
func validateReplicaScaling(cvcOldObj, cvcNewObj *cstor.CStorVolumeConfig) error {
	if isScalingInProgress(cvcOldObj) {
		// if old and new CVC has same count of pools then return true else
		// return false
		if len(cvcOldObj.Spec.Policy.ReplicaPoolInfo) != len(cvcNewObj.Spec.Policy.ReplicaPoolInfo) {
			return errors.Errorf("scaling of CVC %s is already in progress", cvcOldObj.Name)
		}
	}
	return nil
}

// validatePoolNames returns error if there is repeatition of pool names either
// under spec or status of cvc
func validatePoolNames(cvcObj *cstor.CStorVolumeConfig) error {
	replicaPoolNames := cvcObj.GetDesiredReplicaPoolNames()
	// Check repeatition of pool names under Spec of CVC Object
	if !IsUniqueList(replicaPoolNames) {
		return errors.Errorf(
			"duplicate pool names %v found under spec of cvc %s",
			replicaPoolNames,
			cvcObj.Name,
		)
	}
	// Check repeatition of pool names under Status of CVC Object
	if !IsUniqueList(cvcObj.Status.PoolInfo) {
		return errors.Errorf(
			"duplicate pool names %v found under status of cvc %s",
			cvcObj.Status.PoolInfo,
			cvcObj.Name,
		)
	}
	return nil
}

func getCVCObject(name, namespace string,
	clientset clientset.Interface) (*cstor.CStorVolumeConfig, error) {
	return clientset.CstorV1().
		CStorVolumeConfigs(namespace).
		Get(context.TODO(), name, metav1.GetOptions{})
}

// IsUniqueList returns true if values in list are not repeated else return
// false
func IsUniqueList(list []string) bool {
	listMap := map[string]bool{}

	for _, str := range list {
		if _, ok := listMap[str]; ok {
			return false
		}
		listMap[str] = true
	}
	return true
}
