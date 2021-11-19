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
	"testing"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	openebsFakeClientset "github.com/openebs/api/v3/pkg/client/clientset/versioned/fake"
	"github.com/pkg/errors"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type fixture struct {
	wh             *webhook
	openebsObjects []runtime.Object
}

func newFixture() *fixture {
	return &fixture{
		wh: &webhook{},
	}
}

func (f *fixture) withOpenebsObjects(objects ...runtime.Object) *fixture {
	f.openebsObjects = objects
	f.wh.clientset = openebsFakeClientset.NewSimpleClientset(objects...)
	return f
}

func fakeGetCVCError(name, namespace string, clientset clientset.Interface) (*cstor.CStorVolumeConfig, error) {
	return nil, errors.Errorf("fake error")
}

func TestValidateCVCUpdateRequest(t *testing.T) {
	f := newFixture().withOpenebsObjects()
	tests := map[string]struct {
		// existingObj is object existing in etcd via fake client
		existingObj  *cstor.CStorVolumeConfig
		requestedObj *cstor.CStorVolumeConfig
		expectedRsp  bool
		getCVCObj    getCVC
	}{
		"When failed to Get Object From etcd": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc1",
					Namespace: "openebs",
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc1",
					Namespace: "openebs",
				},
				Status: cstor.CStorVolumeConfigStatus{
					Phase: cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: false,
			getCVCObj:   fakeGetCVCError,
		},
		"When ReplicaCount Updated": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc2",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					Phase: cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc2",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 4,
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					Phase: cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: false,
			getCVCObj:   getCVCObject,
		},
		"When Volume Bound Status Updated With Pool Info": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc3",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc3",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool3"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					Phase:    cstor.CStorVolumeConfigPhaseBound,
					PoolInfo: []string{"pool1", "pool2", "pool3"},
				},
			},
			expectedRsp: true,
			getCVCObj:   getCVCObject,
		},
		"When Volume Replicas were Scaled by modifying existing pool names": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc4",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool3"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2", "pool3"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc4",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool5"},
							cstor.ReplicaPoolInfo{PoolName: "pool4"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2", "pool3"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: false,
			getCVCObj:   getCVCObject,
		},
		"When Volume Replicas were migrated": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc5",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool3"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2", "pool3"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc5",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool0"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool5"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2", "pool3"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: false,
			getCVCObj:   getCVCObject,
		},
		"When CVC Scaling Up InProgress Performing Scaling Again": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc6",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool3"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc6",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool3"},
							cstor.ReplicaPoolInfo{PoolName: "pool4"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: false,
			getCVCObj:   getCVCObject,
		},
		"When More Than One Replica Were Scale Down": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc7",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool3"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2", "pool3"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc7",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2", "pool3"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: false,
			getCVCObj:   getCVCObject,
		},
		"When ScaleUp was Performed Before CVC In Bound State": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc8",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc8",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool3"},
						},
					},
				},
			},
			expectedRsp: false,
			getCVCObj:   getCVCObject,
		},
		"When Scale Up Alone Performed": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc10",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc10",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool3"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: true,
			getCVCObj:   getCVCObject,
		},
		"When Scale Down Alone Performed": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc11",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc11",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: true,
			getCVCObj:   getCVCObject,
		},
		"When Scale Up Status Was Updated Success": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc12",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc12",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: true,
			getCVCObj:   getCVCObject,
		},
		"When Scale Down Status Was Updated Success": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc13",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc13",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: true,
			getCVCObj:   getCVCObject,
		},
		"When CVC Spec Pool Names Were Repeated": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc14",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc14",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: false,
			getCVCObj:   getCVCObject,
		},
		"When CVC Status Pool Names Were Repeated": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc15",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool3"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc15",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
							cstor.ReplicaPoolInfo{PoolName: "pool2"},
							cstor.ReplicaPoolInfo{PoolName: "pool3"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1", "pool2", "pool2"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: false,
			getCVCObj:   getCVCObject,
		},
		"When immutable provisioned ReplicaCount has been modified": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc16",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc16",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 2,
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: false,
			getCVCObj:   getCVCObject,
		},
		"When immutable provisioned Capacity has been modified": {
			existingObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc17",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Capacity: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
						Capacity: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			requestedObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cvc17",
					Namespace: "openebs",
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Capacity: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
					Provision: cstor.VolumeProvision{
						ReplicaCount: 1,
						Capacity: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("5Gi"),
						},
					},
					Policy: cstor.CStorVolumePolicySpec{
						ReplicaPoolInfo: []cstor.ReplicaPoolInfo{
							cstor.ReplicaPoolInfo{PoolName: "pool1"},
						},
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					PoolInfo: []string{"pool1"},
					Phase:    cstor.CStorVolumeConfigPhaseBound,
				},
			},
			expectedRsp: false,
			getCVCObj:   getCVCObject,
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			ar := &v1.AdmissionRequest{
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: serialize(test.requestedObj),
				},
			}
			// Create fake object in etcd
			_, err := f.wh.clientset.CstorV1().
				CStorVolumeConfigs(test.existingObj.Namespace).
				Create(context.TODO(), test.existingObj, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf(
					"failed to create fake CVC %s Object in Namespace %s error: %v",
					test.existingObj.Name,
					test.existingObj.Namespace,
					err,
				)
			}
			resp := f.wh.validateCVCUpdateRequest(ar, test.getCVCObj)
			if resp.Allowed != test.expectedRsp {
				t.Errorf(
					"%s test case failed expected response: %t but got %t error: %s",
					name,
					test.expectedRsp,
					resp.Allowed,
					resp.Result.Message,
				)
			}
		})
	}
}
