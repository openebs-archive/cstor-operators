// Copyright 2020 The OpenEBS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package volumemgmt

import (
	"testing"
	"time"

	apis "github.com/openebs/api/pkg/apis/cstor/v1"
	"github.com/openebs/cstor-operators/pkg/controllers/volume-mgmt/volume"

	openebsFakeClientset "github.com/openebs/api/pkg/client/clientset/versioned/fake"

	informers "github.com/openebs/api/pkg/client/informers/externalversions"

	"github.com/openebs/api/pkg/util"
	"github.com/openebs/api/pkg/util/signals"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

// TestRun is to run cStorVolume controller and check if it crashes or return back.
func TestRun(t *testing.T) {
	fakeKubeClient := fake.NewSimpleClientset()
	fakeOpenebsClient := openebsFakeClientset.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(fakeKubeClient, time.Second*30)
	openebsInformerFactory := informers.NewSharedInformerFactory(fakeOpenebsClient, time.Second*30)

	// Instantiate the cStor Volume controllers.
	volumeController := NewCStorVolumeController(fakeKubeClient, fakeOpenebsClient, kubeInformerFactory,
		openebsInformerFactory)

	stopCh := signals.SetupSignalHandler()
	done := make(chan bool)
	go func(chan bool) {
		volumeController.Run(2, stopCh)
		done <- true
	}(done)

	select {
	case <-time.After(3 * time.Second):

	case <-done:
		t.Fatalf("CStorVolume controller returned - failure")

	}
}

// TestProcessNextWorkItemAdd is to test a cStorVolume resource for add event.
func TestProcessNextWorkItemAdd(t *testing.T) {
	fakeKubeClient := fake.NewSimpleClientset()
	fakeOpenebsClient := openebsFakeClientset.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(fakeKubeClient, time.Second*30)
	openebsInformerFactory := informers.NewSharedInformerFactory(fakeOpenebsClient, time.Second*30)

	// Instantiate the cStor Volume controllers.
	volumeController := NewCStorVolumeController(fakeKubeClient, fakeOpenebsClient, kubeInformerFactory,
		openebsInformerFactory)

	testVolumeResource := map[string]struct {
		expectedOutput bool
		test           *apis.CStorVolume
	}{
		"img2VolumeResource": {
			expectedOutput: true,
			test: &apis.CStorVolume{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "volume2",
					UID:        types.UID("abcd"),
					Finalizers: []string{"cstorvolume.openebs.io/finalizer"},
				},
				Spec: apis.CStorVolumeSpec{
					TargetIP: "0.0.0.0",
					Capacity: fakeStrToQuantity("5G"),
					//Status:   "init",
				},
				Status: apis.CStorVolumeStatus{Phase: "init"},
			},
		},
	}
	_, err := volumeController.clientset.CstorV1().CStorVolumes("default").Create(testVolumeResource["img2VolumeResource"].test)
	if err != nil {
		t.Fatalf("Unable to create resource : %v", testVolumeResource["img2VolumeResource"].test.ObjectMeta.Name)
	}

	var q QueueLoad
	q.Key = "volume2"
	q.Operation = QOpAdd
	volumeController.workqueue.AddRateLimited(q)
	volume.FileOperatorVar = util.TestFileOperator{}
	volume.UnixSockVar = util.TestUnixSock{}
	obtainedOutput := volumeController.processNextWorkItem()
	if obtainedOutput != testVolumeResource["img2VolumeResource"].expectedOutput {
		t.Fatalf("Expected:%v, Got:%v", testVolumeResource["img2VolumeResource"].expectedOutput,
			obtainedOutput)
	}
}

// TestProcessNextWorkItemModify is to test a cStorVolume resource for modify event.
func TestProcessNextWorkItemModify(t *testing.T) {
	fakeKubeClient := fake.NewSimpleClientset()
	fakeOpenebsClient := openebsFakeClientset.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(fakeKubeClient, time.Second*30)
	openebsInformerFactory := informers.NewSharedInformerFactory(fakeOpenebsClient, time.Second*30)

	// Instantiate the cStor Volume controllers.
	volumeController := NewCStorVolumeController(fakeKubeClient, fakeOpenebsClient, kubeInformerFactory,
		openebsInformerFactory)

	testVolumeResource := map[string]struct {
		expectedOutput bool
		test           *apis.CStorVolume
	}{
		"img2VolumeResource": {
			expectedOutput: true,
			test: &apis.CStorVolume{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "volume2",
					UID:        types.UID("abcd"),
					Finalizers: []string{"cstorvolume.openebs.io/finalizer"},
				},
				Spec: apis.CStorVolumeSpec{
					TargetIP: "0.0.0.0",
					Capacity: fakeStrToQuantity("5G"),
					//Status:   "init",
				},
				Status: apis.CStorVolumeStatus{},
			},
		},
	}

	_, err := volumeController.clientset.CstorV1().CStorVolumes("default").Create(testVolumeResource["img2VolumeResource"].test)
	if err != nil {
		t.Fatalf("Unable to create resource : %v", testVolumeResource["img2VolumeResource"].test.ObjectMeta.Name)
	}

	var q QueueLoad
	q.Key = "volume2"
	q.Operation = QOpModify
	volumeController.workqueue.AddRateLimited(q)

	obtainedOutput := volumeController.processNextWorkItem()
	if obtainedOutput != testVolumeResource["img2VolumeResource"].expectedOutput {
		t.Fatalf("Expected:%v, Got:%v", testVolumeResource["img2VolumeResource"].expectedOutput,
			obtainedOutput)
	}
}

// TestProcessNextWorkItemDestroy is to test a cStorVolume resource for destroy event.
func TestProcessNextWorkItemDestroy(t *testing.T) {
	fakeKubeClient := fake.NewSimpleClientset()
	fakeOpenebsClient := openebsFakeClientset.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(fakeKubeClient, time.Second*30)
	openebsInformerFactory := informers.NewSharedInformerFactory(fakeOpenebsClient, time.Second*30)

	// Instantiate the cStor Volume controllers.
	volumeController := NewCStorVolumeController(fakeKubeClient, fakeOpenebsClient, kubeInformerFactory,
		openebsInformerFactory)

	testVolumeResource := map[string]struct {
		expectedOutput bool
		test           *apis.CStorVolume
	}{
		"img2VolumeResource": {
			expectedOutput: true,
			test: &apis.CStorVolume{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "volume2",
					UID:        types.UID("abcd"),
					Finalizers: []string{"cstorvolume.openebs.io/finalizer"},
				},
				Spec: apis.CStorVolumeSpec{
					TargetIP: "0.0.0.0",
					Capacity: fakeStrToQuantity("5G"),
					//Status:   "init",
				},
				Status: apis.CStorVolumeStatus{},
			},
		},
	}

	_, err := volumeController.clientset.CstorV1().CStorVolumes("default").Create(testVolumeResource["img2VolumeResource"].test)
	if err != nil {
		t.Fatalf("Unable to create resource : %v", testVolumeResource["img2VolumeResource"].test.ObjectMeta.Name)
	}

	var q QueueLoad
	q.Key = "volume2"
	q.Operation = QOpDestroy
	volumeController.workqueue.AddRateLimited(q)

	obtainedOutput := volumeController.processNextWorkItem()
	if obtainedOutput != testVolumeResource["img2VolumeResource"].expectedOutput {
		t.Fatalf("Expected:%v, Got:%v", testVolumeResource["img2VolumeResource"].expectedOutput,
			obtainedOutput)
	}
}
