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

package cspccontroller

import (
	"context"

	apis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"

	"github.com/openebs/api/v3/pkg/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// cleanupCSPIResources removes the CSPI resources when a CSPI is
// deleted or downscaled
func (c *Controller) cleanupCSPIResources(cspiList *apis.CStorPoolInstanceList) error {

	opts := []cspiCleanupOptions{
		c.cleanupBDC,
	}

	var cspiCleanUpError []error

	for _, cspiItem := range cspiList.Items {
		cspiObj := cspiItem // pin it
		// cleanup to be performed only if DeletionTimestamp is non zero and if
		// PoolProtectionFinalizer is not removed wait for the next reconcile attempt
		if canPerformCSPICleanup(cspiItem) {
			for _, o := range opts {
				err := o(cspiObj)
				if err != nil {
					return errors.Wrapf(err, "failed to cleanup cspi %s", cspiItem.Name)
				}
			}

			cspiObj.Finalizers = util.RemoveString(cspiObj.Finalizers, types.CSPCFinalizer)
			_, err := c.GetStoredCStorVersionClient().CStorPoolInstances(cspiItem.Namespace).Update(context.TODO(), &cspiObj, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to remove finalizer from cspi %s", cspiItem.Name)
			}
			klog.Infof("cleanup for cspi %s was successful", cspiItem.Name)
		} else {
			if isDestroyed(cspiItem) {
				// if cspi has DeletionTimestamp but the PoolProtectionFinalizer is present
				// returning error helps prevent removal of finalizer on cspc object
				// cspc object should not get deleted before all cspi are deleted successfully
				newErr := errors.Errorf("failed to cleanup cspi %s: waiting for pool to get destroyed or there is no CSPC label on CSPI",
					cspiItem.Name)
				cspiCleanUpError = append(cspiCleanUpError, newErr)
			}
		}
	}
	if len(cspiCleanUpError) > 0 {
		return errors.Errorf("failure in cspi cleanup: {%v}", cspiCleanUpError)
	}

	return nil
}

// canPerformCSPICleanup performs the validation if the cleanup for the
// CSPI can begin
func canPerformCSPICleanup(cspiObj apis.CStorPoolInstance) bool {
	predicates := []cspiCleanupPredicates{
		isDestroyed,
		hasCSPCFinalizer,
		hasNoPoolProtectionFinalizer,
	}
	for _, p := range predicates {
		if !p(cspiObj) {
			return false
		}
	}
	return true
}

type cspiCleanupPredicates func(apis.CStorPoolInstance) bool

// isDestroyed is to check if the call is for cStorPoolInstance destroy.
func isDestroyed(cspiObj apis.CStorPoolInstance) bool {
	return !cspiObj.DeletionTimestamp.IsZero()
}

// hasCSPCFinalizer is a predicate which checks whether the CSPC
// finalizer is presemt on the CSPI or not
func hasCSPCFinalizer(cspiObj apis.CStorPoolInstance) bool {
	return util.ContainsString(cspiObj.Finalizers, types.CSPCFinalizer)
}

// hasNoPoolProtectionFinalizer is a predicate which checks whether the pool
// protection finalizer is removed or not. The pool protection finalizer is
// used to make sure that the pool is destroyed before BDCs are deleted.
func hasNoPoolProtectionFinalizer(cspiObj apis.CStorPoolInstance) bool {
	return !util.ContainsString(cspiObj.Finalizers, types.PoolProtectionFinalizer)
}

type cspiCleanupOptions func(apis.CStorPoolInstance) error

// cleanupBDC deletes the BDCs for the CSPI which has been deleted or downscaled
func (c *Controller) cleanupBDC(cspiObj apis.CStorPoolInstance) error {
	bdcList, err := c.GetStoredOpenebsVersionClient().BlockDeviceClaims(cspiObj.Namespace).List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: string(types.CStorPoolClusterLabelKey) + "=" + cspiObj.Labels[string(types.CStorPoolClusterLabelKey)],
		},
	)
	if err != nil {
		return err
	}
	cspiBDMap := map[string]bool{}
	for _, raidGroup := range cspiObj.Spec.DataRaidGroups {
		for _, bdcObj := range raidGroup.CStorPoolInstanceBlockDevices {
			cspiBDMap[bdcObj.BlockDeviceName] = true
		}
	}
	for _, bdcItem := range bdcList.Items {
		bdcItem := bdcItem // pin it
		if cspiBDMap[bdcItem.Spec.BlockDeviceName] {
			bdcObj := &bdcItem
			bdcObj.Finalizers = util.RemoveString(bdcObj.Finalizers, types.CSPCFinalizer)
			bdcObj, err = c.GetStoredOpenebsVersionClient().BlockDeviceClaims(cspiObj.Namespace).Update(context.TODO(), bdcObj, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to remove finalizers from bdc %s", bdcItem.Name)
			}
			err = c.GetStoredOpenebsVersionClient().BlockDeviceClaims(cspiObj.Namespace).Delete(context.TODO(), bdcObj.Name, metav1.DeleteOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to delete bdc %s", bdcObj.Name)
			}
		}
	}
	return err
}
