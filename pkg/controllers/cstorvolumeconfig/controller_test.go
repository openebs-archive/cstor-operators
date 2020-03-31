/*
Copyright 2019 The OpenEBS Authors

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

package cstorvolumeconfig

import (
	"reflect"
	"testing"
	"time"

	apis "github.com/openebs/api/pkg/apis/cstor/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type conditionMergeTestCase struct {
	description    string
	cvc            *apis.CStorVolumeConfig
	newConditions  []apis.CStorVolumeConfigCondition
	finalCondtions []apis.CStorVolumeConfigCondition
}

func TestMergeResizeCondition(t *testing.T) {
	currentTime := metav1.Now()

	cvc := getCVC([]apis.CStorVolumeConfigCondition{
		{
			Type:               apis.CStorVolumeConfigResizing,
			LastTransitionTime: currentTime,
		},
	})

	noConditionCVC := getCVC([]apis.CStorVolumeConfigCondition{})

	conditionFalseTime := metav1.Now()
	newTime := metav1.NewTime(time.Now().Add(1 * time.Hour))

	testCases := []conditionMergeTestCase{
		{
			description:    "when removing all conditions",
			cvc:            cvc.DeepCopy(),
			newConditions:  []apis.CStorVolumeConfigCondition{},
			finalCondtions: []apis.CStorVolumeConfigCondition{},
		},
		{
			description: "adding new condition",
			cvc:         cvc.DeepCopy(),
			newConditions: []apis.CStorVolumeConfigCondition{
				{
					Type: apis.CStorVolumeConfigResizePending,
				},
			},
			finalCondtions: []apis.CStorVolumeConfigCondition{
				{
					Type: apis.CStorVolumeConfigResizePending,
				},
			},
		},
		{
			description: "adding same condition with new timestamp",
			cvc:         cvc.DeepCopy(),
			newConditions: []apis.CStorVolumeConfigCondition{
				{
					Type:               apis.CStorVolumeConfigResizing,
					LastTransitionTime: newTime,
				},
			},
			finalCondtions: []apis.CStorVolumeConfigCondition{
				{
					Type:               apis.CStorVolumeConfigResizing,
					LastTransitionTime: newTime,
				},
			},
		},
		{
			description: "adding same condition but with different status",
			cvc:         cvc.DeepCopy(),
			newConditions: []apis.CStorVolumeConfigCondition{
				{
					Type:               apis.CStorVolumeConfigResizing,
					LastTransitionTime: conditionFalseTime,
				},
			},
			finalCondtions: []apis.CStorVolumeConfigCondition{
				{
					Type:               apis.CStorVolumeConfigResizing,
					LastTransitionTime: conditionFalseTime,
				},
			},
		},
		{
			description: "when no condition exists on pvc",
			cvc:         noConditionCVC.DeepCopy(),
			newConditions: []apis.CStorVolumeConfigCondition{
				{
					Type:               apis.CStorVolumeConfigResizing,
					LastTransitionTime: currentTime,
				},
			},
			finalCondtions: []apis.CStorVolumeConfigCondition{
				{
					Type:               apis.CStorVolumeConfigResizing,
					LastTransitionTime: currentTime,
				},
			},
		},
	}

	for _, testcase := range testCases {
		updateConditions := MergeResizeConditionsOfCVC(testcase.cvc.Status.Conditions, testcase.newConditions)

		if !reflect.DeepEqual(updateConditions, testcase.finalCondtions) {
			t.Errorf("Expected updated conditions for test %s to be %v but got %v",
				testcase.description,
				testcase.finalCondtions, updateConditions)
		}
	}

}

func getCVC(conditions []apis.CStorVolumeConfigCondition) *apis.CStorVolumeConfig {
	cvc := &apis.CStorVolumeConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "openebs"},
		Spec: apis.CStorVolumeConfigSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("2Gi"),
			},
		},
		Status: apis.CStorVolumeConfigStatus{
			Phase:      apis.CStorVolumeConfigPhaseBound,
			Conditions: conditions,
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("2Gi"),
			},
		},
	}
	return cvc
}
