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
	"testing"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebsFakeClientset "github.com/openebs/api/v3/pkg/client/clientset/versioned/fake"
	cspiutil "github.com/openebs/cstor-operators/pkg/controllers/cspi-controller/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

// NOTE: Since we are running below test in parallel maintain unique CSPI names
func TestUpdateStatusConditionEventually(t *testing.T) {
	controller := CStorPoolInstanceController{
		clientset: openebsFakeClientset.NewSimpleClientset(),
		// NewFakeRecorder creates new fake event
		// recorder with event channel with buffer of given size
		recorder: record.NewFakeRecorder(5),
	}
	time := metav1.Now()
	tests := map[string]struct {
		cspi                    *cstor.CStorPoolInstance
		newCondition            cstor.CStorPoolInstanceCondition
		isObjectNeedToBeCreated bool
		isErrorExpected         bool
		expectedCondLength      int
		expectedTransistionTime metav1.Time
	}{
		"When there are no condition condition new condition should be addded": {
			cspi: &cstor.CStorPoolInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cstor-pool-1",
					Namespace: "openebs",
				},
			},
			newCondition: cstor.CStorPoolInstanceCondition{
				Type:               cstor.CSPIPoolExpansion,
				Status:             corev1.ConditionTrue,
				Reason:             "PoolExpansionInProgress",
				Message:            "triggered pool expansion",
				LastUpdateTime:     metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			isObjectNeedToBeCreated: true,
			expectedCondLength:      1,
			isErrorExpected:         false,
		},
		"When existing condition message in CSPI got updated": {
			cspi: &cstor.CStorPoolInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cstor-pool-2",
					Namespace: "openebs",
				},
				Status: cstor.CStorPoolInstanceStatus{
					Conditions: []cstor.CStorPoolInstanceCondition{
						cstor.CStorPoolInstanceCondition{
							Type:               cstor.CSPIPoolExpansion,
							Status:             corev1.ConditionTrue,
							Reason:             "PoolExpansionInProgress",
							Message:            "triggered pool expansion",
							LastUpdateTime:     time,
							LastTransitionTime: time,
						},
					},
				},
			},
			newCondition: cstor.CStorPoolInstanceCondition{
				Type:               cstor.CSPIPoolExpansion,
				Status:             corev1.ConditionTrue,
				Reason:             "PoolExpansionInProgress",
				Message:            "error: failed to add vdev in CSPI",
				LastUpdateTime:     metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			isObjectNeedToBeCreated: true,
			expectedCondLength:      1,
			isErrorExpected:         false,
			expectedTransistionTime: time,
		},
		"When new condition got added": {
			cspi: &cstor.CStorPoolInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cstor-pool-3",
					Namespace: "openebs",
				},
				Status: cstor.CStorPoolInstanceStatus{
					Conditions: []cstor.CStorPoolInstanceCondition{
						cstor.CStorPoolInstanceCondition{
							Type:               cstor.CSPIPoolExpansion,
							Status:             corev1.ConditionTrue,
							Reason:             "PoolExpansionInProgress",
							Message:            "triggered pool expansion",
							LastUpdateTime:     metav1.Now(),
							LastTransitionTime: metav1.Now(),
						},
					},
				},
			},
			newCondition: cstor.CStorPoolInstanceCondition{
				Type:               cstor.CSPIDiskReplacement,
				Status:             corev1.ConditionTrue,
				Reason:             "DiskReplacementInProgress",
				Message:            "triggered disk replacement",
				LastUpdateTime:     metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			isObjectNeedToBeCreated: true,
			expectedCondLength:      2,
			isErrorExpected:         false,
		},
		"When we not able to fetch the object from etcd": {
			cspi: &cstor.CStorPoolInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cstor-pool-4",
					Namespace: "openebs",
				},
			},
			newCondition: cstor.CStorPoolInstanceCondition{
				Type:               cstor.CSPIPoolExpansion,
				Status:             corev1.ConditionTrue,
				Reason:             "PoolExpansionInProgress",
				Message:            "error: failed to add vdev in CSPI",
				LastUpdateTime:     metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			isObjectNeedToBeCreated: false,
			expectedCondLength:      0,
			isErrorExpected:         true,
		},
		"When there are multiple conditions and status got updated to true": {
			cspi: &cstor.CStorPoolInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cstor-pool-5",
					Namespace: "openebs",
				},
				Status: cstor.CStorPoolInstanceStatus{
					Conditions: []cstor.CStorPoolInstanceCondition{
						cstor.CStorPoolInstanceCondition{
							Type:               cstor.CSPIPoolExpansion,
							Status:             corev1.ConditionTrue,
							Reason:             "PoolExpansionInProgress",
							Message:            "triggered pool expansion",
							LastUpdateTime:     time,
							LastTransitionTime: time,
						},
						cstor.CStorPoolInstanceCondition{
							Type:               cstor.CSPIDiskReplacement,
							Status:             corev1.ConditionTrue,
							Reason:             "BlockDeviceReplacementInProgress",
							Message:            "triggered disk replacement",
							LastUpdateTime:     time,
							LastTransitionTime: time,
						},
					},
				},
			},
			newCondition: cstor.CStorPoolInstanceCondition{
				Type:               cstor.CSPIPoolExpansion,
				Status:             corev1.ConditionFalse,
				Reason:             "PoolExpansionSuccessfull",
				Message:            "",
				LastUpdateTime:     metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			isObjectNeedToBeCreated: true,
			expectedCondLength:      2,
			isErrorExpected:         false,
		},
	}

	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			// Create object only if isObjectNeedToBeCreated
			if test.isObjectNeedToBeCreated {
				_, err := controller.clientset.
					CstorV1().
					CStorPoolInstances(test.cspi.Namespace).
					Create(context.TODO(), test.cspi, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("Failed to create fake object: %v", err)
				}
			}
			updatedCSPI, err := controller.UpdateStatusConditionEventually(test.cspi, test.newCondition)
			if test.isErrorExpected && err == nil {
				t.Fatalf("Test: %q failed expected err not to be nil", name)
			} else if !test.isErrorExpected && err != nil {
				t.Fatalf("Test: %q failed expected err to be nil but got %v", name, err)
			} else {
				if !test.isErrorExpected {
					if len(updatedCSPI.Status.Conditions) != test.expectedCondLength {
						t.Fatalf("Test %q failed: expected no.of conditions %d but got %d",
							name, test.expectedCondLength, len(updatedCSPI.Status.Conditions),
						)
					}
					condition := cspiutil.GetCSPICondition(updatedCSPI.Status, test.newCondition.Type)
					if condition == nil {
						t.Fatalf("Test %q failed expected %s condition type to exist but it doesn't exist %v",
							name, test.newCondition.Type, updatedCSPI.Status.Conditions)
					}
					if !test.expectedTransistionTime.IsZero() && !test.expectedTransistionTime.Equal(&condition.LastTransitionTime) {
						t.Fatalf("Test %q failed expected LastTransistion %v but got updated to %v",
							name, test.expectedTransistionTime, condition.LastTransitionTime)
					}
				}
			}
		})
	}
}
