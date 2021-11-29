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
	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	openebsFakeClientset "github.com/openebs/api/v3/pkg/client/clientset/versioned/fake"
	informers "github.com/openebs/api/v3/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"

	"testing"
	"time"
)

var (
	alwaysReady = func() bool { return true }
)

func newCSPI(name string, labels map[string]string, finalizers []string, deletionTimeStamp bool) *cstor.CStorPoolInstance {
	cspi := &cstor.CStorPoolInstance{}
	cspi.Name = name
	cspi.Labels = labels
	cspi.Finalizers = finalizers
	if deletionTimeStamp {
		time := metav1.Now()
		cspi.DeletionTimestamp = &time
	}
	return cspi
}

// Returns 0 for resyncPeriod in case resyncing is not needed.
func NoResyncPeriodFunc() time.Duration {
	return 0
}

func TestController_cleanupCSPIResources(t *testing.T) {
	tests := []struct {
		cspis             *cstor.CStorPoolInstanceList
		expectedDeletions int
	}{
		{
			cspis: &cstor.CStorPoolInstanceList{
				Items: []cstor.CStorPoolInstance{
					*newCSPI("cspi-foo", nil, []string{types.CSPCFinalizer}, true),
				},
			},
			expectedDeletions: 1,
		},

		{
			cspis: &cstor.CStorPoolInstanceList{
				Items: []cstor.CStorPoolInstance{
					*newCSPI("cspi-foo", nil, []string{types.CSPCFinalizer}, false),
					*newCSPI("cspi-bar", nil, []string{}, true),
					*newCSPI("cspi-ki", nil, []string{types.CSPCFinalizer, types.PoolProtectionFinalizer}, true),
					*newCSPI("cspi-ka", nil, []string{types.CSPCFinalizer}, true),
				},
			},
			expectedDeletions: 1,
		},

		{
			cspis: &cstor.CStorPoolInstanceList{
				Items: []cstor.CStorPoolInstance{
					*newCSPI("cspi-foo", nil, []string{types.CSPCFinalizer}, true),
					*newCSPI("cspi-bar", nil, []string{types.CSPCFinalizer}, true),
					*newCSPI("cspi-xoo", nil, []string{types.CSPCFinalizer}, true),
				},
			},
			expectedDeletions: 3,
		},
	}

	for i := range tests {
		test := tests[i]
		t.Logf("scenario %d", i)

		fakeKubeClientSet := &fake.Clientset{}
		fakeOpenEBSClientSet := &openebsFakeClientset.Clientset{}
		kubeInformerFactory := kubeinformers.NewSharedInformerFactory(fakeKubeClientSet, NoResyncPeriodFunc())
		cspcInformerFactory := informers.NewSharedInformerFactory(fakeOpenEBSClientSet, NoResyncPeriodFunc())

		controller, err := NewControllerBuilder().
			WithKubeClient(fakeKubeClientSet).
			WithOpenEBSClient(fakeOpenEBSClientSet).
			WithCSPCLister(cspcInformerFactory).
			WithEventHandler(cspcInformerFactory).
			WithWorkqueueRateLimiting().Build()

		if err != nil {
			t.Fatalf("error creating Deployment controller: %v", err)
		}

		controller.recorder = &record.FakeRecorder{}
		controller.cspcSynced = alwaysReady
		controller.cspiSynced = alwaysReady

		for _, cspi := range test.cspis.Items {
			cspcInformerFactory.Cstor().V1().CStorPoolInstances().Informer().GetIndexer().Add(cspi)
		}

		stopCh := make(chan struct{})
		defer close(stopCh)
		kubeInformerFactory.Start(stopCh)
		cspcInformerFactory.Start(stopCh)

		controller.cleanupCSPIResources(test.cspis)

		gotDeletions := 0
		for _, action := range fakeOpenEBSClientSet.Actions() {
			if action.Matches("update", "cstorpoolinstances") {
				gotDeletions++
			}
		}

		if gotDeletions != test.expectedDeletions {
			t.Errorf("expect %v cspis been deleted, but got %v", test.expectedDeletions, gotDeletions)
			continue
		}
	}
}
