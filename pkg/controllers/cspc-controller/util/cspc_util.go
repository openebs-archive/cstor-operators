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

package util

import (
	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PoolManagersAvailable is added in a cspc when it has its minimum pool-managers required available.
	MinimumPoolManagersAvailable = "MinimumPoolManagersAvailable"
	// MinimumPoolManagersUnAvailable is added in a cspc when it doesn't have the minimum required pool-managers
	// available.
	MinimumPoolManagersUnAvailable = "MinimumPoolManagersUnAvailable"
)

// ToDo: Move this to openebs/api once the conditions and status OEP is merged.
// Reference: https://github.com/openebs/openebs/pull/2942

const (
	// PoolManagerAvailable is
	PoolManagerAvailable cstor.CSPCConditionType = "PoolManagerAvailable"
)

// NewCSPCCondition creates a new cspc condition.
func NewCSPCCondition(condType cstor.CSPCConditionType, status corev1.ConditionStatus, reason, message string) *cstor.CStorPoolClusterCondition {
	return &cstor.CStorPoolClusterCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetCSPCCondition returns the condition with the provided type.
func GetCSPCCondition(status cstor.CStorPoolClusterStatus, condType cstor.CSPCConditionType) *cstor.CStorPoolClusterCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCSPCCondition updates the cspc to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update.
func SetCSPCCondition(status *cstor.CStorPoolClusterStatus, condition cstor.CStorPoolClusterCondition) {
	currentCond := GetCSPCCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}
	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// RemoveCSPCCondition removes the cspc condition with the provided type.
func RemoveCSPCCondition(status *cstor.CStorPoolClusterStatus, condType cstor.CSPCConditionType) {
	status.Conditions = filterOutCondition(status.Conditions, condType)
}

// filterOutCondition returns a new slice of cspc conditions without conditions with the provided type.
func filterOutCondition(conditions []cstor.CStorPoolClusterCondition, condType cstor.CSPCConditionType) []cstor.CStorPoolClusterCondition {
	var newConditions []cstor.CStorPoolClusterCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
