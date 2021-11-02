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

package cspicontroller

import (
	"context"
	"time"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	cspiutil "github.com/openebs/cstor-operators/pkg/controllers/cspi-controller/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

//TODO: Update the code to use patch instead of Update call

// UpdateStatusConditionEventually updates the CSPI in etcd with provided
// condition. Below function retries for three times to update the CSPI with
// provided conditions
func (c *CStorPoolInstanceController) UpdateStatusConditionEventually(
	cspi *cstor.CStorPoolInstance,
	condition cstor.CStorPoolInstanceCondition) (*cstor.CStorPoolInstance, error) {
	maxRetry := 3
	cspiCopy := cspi.DeepCopy()
	updatedCSPI, err := c.UpdateStatusCondition(cspiCopy, condition)
	if err != nil {
		klog.Errorf(
			"failed to update CSPI %s status with condition %s will retry %d times at 2s interval: {%s}",
			cspi.Name, condition.Type, maxRetry, err.Error())

		for maxRetry > 0 {
			newCSPI, err := c.clientset.
				CstorV1().
				CStorPoolInstances(cspi.Namespace).
				Get(context.TODO(), cspi.Name, metav1.GetOptions{})
			if err != nil {
				// This is possible due to etcd unavailability so do not retry more here
				return cspi, errors.Wrapf(err, "failed to update cspi status")
			}
			updatedCSPI, err = c.UpdateStatusCondition(newCSPI, condition)
			if err != nil {
				maxRetry = maxRetry - 1
				klog.Errorf(
					"failed to update CSPI %s status with condition %s will retry %d times at 2s interval: {%s}",
					cspi.Name, condition.Type, maxRetry, err.Error())
				time.Sleep(2 * time.Second)
				continue
			}
			return updatedCSPI, nil
		}
		// When retries are completed and still failed to update in etcd
		// then it will return original object
		return cspi, err
	}
	return updatedCSPI, nil
}

func (c *CStorPoolInstanceController) UpdateStatusCondition(
	cspi *cstor.CStorPoolInstance,
	condition cstor.CStorPoolInstanceCondition) (*cstor.CStorPoolInstance, error) {
	cspiutil.SetCSPICondition(&cspi.Status, condition)
	updatedCSPI, err := c.clientset.
		CstorV1().
		CStorPoolInstances(cspi.Namespace).
		Update(context.TODO(), cspi, metav1.UpdateOptions{})
	if err != nil {
		// cspi object has already updated with the conditions so returning
		// same object may or maynot make sense
		return nil, errors.Wrapf(err, "failed to update cspi conditions")
	}
	return updatedCSPI, nil
}
