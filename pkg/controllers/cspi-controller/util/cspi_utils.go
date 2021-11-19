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

package cspicontroller

import (
	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCSPICondition creates a new cspi condition.
func NewCSPICondition(condType cstor.CStorPoolInstanceConditionType, status corev1.ConditionStatus, reason, message string) *cstor.CStorPoolInstanceCondition {
	return &cstor.CStorPoolInstanceCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetCSPICondition returns the condition with the provided type.
func GetCSPICondition(
	status cstor.CStorPoolInstanceStatus,
	condType cstor.CStorPoolInstanceConditionType) *cstor.CStorPoolInstanceCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCSPICondition updates the cspi to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update.
func SetCSPICondition(status *cstor.CStorPoolInstanceStatus, condition cstor.CStorPoolInstanceCondition) {
	currentCond := GetCSPICondition(*status, condition.Type)
	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Type == condition.Type && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// RemoveCSPICondition removes the cspi condition with the provided type.
func RemoveCSPICondition(status *cstor.CStorPoolInstanceStatus, condType cstor.CStorPoolInstanceConditionType) {
	status.Conditions = filterOutCondition(status.Conditions, condType)
}

// filterOutCondition returns a new slice of cspi conditions without conditions with the provided type.
func filterOutCondition(conditions []cstor.CStorPoolInstanceCondition,
	condType cstor.CStorPoolInstanceConditionType) []cstor.CStorPoolInstanceCondition {
	var newConditions []cstor.CStorPoolInstanceCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
