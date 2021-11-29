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
	"reflect"

	"testing"
)

var (
	condPoolManagerAvailable = func() cstor.CStorPoolClusterCondition {
		return cstor.CStorPoolClusterCondition{
			Type:   PoolManagerAvailable,
			Status: corev1.ConditionTrue,
			Reason: "AwesomeCSPCController",
		}
	}

	condPoolManagerImaginary = func() cstor.CStorPoolClusterCondition {
		return cstor.CStorPoolClusterCondition{
			Type:   "Imaginary",
			Status: corev1.ConditionTrue,
			Reason: "AwesomeCSPCController",
		}
	}

	condPoolManagerImaginary1 = func() cstor.CStorPoolClusterCondition {
		return cstor.CStorPoolClusterCondition{
			Type:   "Imaginary",
			Status: corev1.ConditionFalse,
			Reason: "ForSomeReason",
		}
	}

	status = func() *cstor.CStorPoolClusterStatus {
		return &cstor.CStorPoolClusterStatus{
			Conditions: []cstor.CStorPoolClusterCondition{condPoolManagerImaginary(), condPoolManagerAvailable()},
		}
	}
)

func TestGetCondition(t *testing.T) {
	exampleStatus := status()

	tests := []struct {
		name string

		status   cstor.CStorPoolClusterStatus
		condType cstor.CSPCConditionType

		expected bool
	}{
		{
			name:     "condition exists",
			status:   *exampleStatus,
			condType: PoolManagerAvailable,
			expected: true,
		},
		{
			name:     "condition does not exist",
			status:   *exampleStatus,
			condType: "testCondition",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cond := GetCSPCCondition(test.status, test.condType)
			exists := cond != nil
			if exists != test.expected {
				t.Errorf("%s: expected condition to exist: %t, got: %t", test.name, test.expected, exists)
			}
		})
	}
}

func TestSetCondition(t *testing.T) {
	tests := []struct {
		name string

		status *cstor.CStorPoolClusterStatus
		cond   cstor.CStorPoolClusterCondition

		expectedStatus *cstor.CStorPoolClusterStatus
	}{
		{
			name:           "set for the first time",
			status:         &cstor.CStorPoolClusterStatus{},
			cond:           condPoolManagerAvailable(),
			expectedStatus: &cstor.CStorPoolClusterStatus{Conditions: []cstor.CStorPoolClusterCondition{condPoolManagerAvailable()}},
		},
		{
			name:           "simple set",
			status:         &cstor.CStorPoolClusterStatus{Conditions: []cstor.CStorPoolClusterCondition{condPoolManagerImaginary()}},
			cond:           condPoolManagerAvailable(),
			expectedStatus: status(),
		},
		{
			name:           "overwrite",
			status:         &cstor.CStorPoolClusterStatus{Conditions: []cstor.CStorPoolClusterCondition{condPoolManagerImaginary()}},
			cond:           condPoolManagerImaginary1(),
			expectedStatus: &cstor.CStorPoolClusterStatus{Conditions: []cstor.CStorPoolClusterCondition{condPoolManagerImaginary1()}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			SetCSPCCondition(test.status, test.cond)
			if !reflect.DeepEqual(test.status, test.expectedStatus) {
				t.Errorf("%s: expected status: %v, got: %v", test.name, test.expectedStatus, test.status)
			}
		})
	}
}

func TestRemoveCondition(t *testing.T) {
	tests := []struct {
		name string

		status   *cstor.CStorPoolClusterStatus
		condType cstor.CSPCConditionType

		expectedStatus *cstor.CStorPoolClusterStatus
	}{
		{
			name:           "remove from empty status",
			status:         &cstor.CStorPoolClusterStatus{},
			condType:       PoolManagerAvailable,
			expectedStatus: &cstor.CStorPoolClusterStatus{},
		},
		{
			name:           "simple remove",
			status:         &cstor.CStorPoolClusterStatus{Conditions: []cstor.CStorPoolClusterCondition{condPoolManagerImaginary()}},
			condType:       "Imaginary",
			expectedStatus: &cstor.CStorPoolClusterStatus{},
		},
		{
			name:           "doesn't remove anything",
			status:         status(),
			condType:       "test-condition",
			expectedStatus: status(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RemoveCSPCCondition(test.status, test.condType)
			if !reflect.DeepEqual(test.status, test.expectedStatus) {
				t.Errorf("%s: expected status: %v, got: %v", test.name, test.expectedStatus, test.status)
			}
		})
	}
}
