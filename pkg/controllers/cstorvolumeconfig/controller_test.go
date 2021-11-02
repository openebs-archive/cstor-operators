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
	"context"
	"fmt"
	"html"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	apis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	apistypes "github.com/openebs/api/v3/pkg/apis/types"
	openebsFakeClientset "github.com/openebs/api/v3/pkg/client/clientset/versioned/fake"
	openebsinformers "github.com/openebs/api/v3/pkg/client/informers/externalversions"
	"github.com/openebs/cstor-operators/pkg/controllers/testutil"
	errors "github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	alwaysReady = func() bool { return true }
	namespace   = "openebs"
)

// fixture encapsulates fake client sets and client-go testing objects.
// This is useful in mocking a controller.
type fixture struct {
	t *testing.T
	// k8sClient is the fake client set for k8s native objects.
	k8sClient *fake.Clientset
	// openebsClient is the fake client set for openebs cr objects.
	openebsClient *openebsFakeClientset.Clientset

	// Objects to put in the store.
	cvcLister  []*apis.CStorVolumeConfig
	cspiLister []*apis.CStorPoolInstance
	cvrLister  []*apis.CStorVolumeReplica
	cvLister   []*apis.CStorVolume

	// Actions expected to happen on the client. Objects from here are also
	// preloaded into NewSimpleFake.
	k8sObjects     []runtime.Object
	openebsObjects []runtime.Object
}

// testConfig contains the extra information required to run the test
type testConfig struct {
	// loopCount times reconcile function will be called
	loopCount int
	// time interval to trigger reconciliation
	loopDelay time.Duration
}

// newFixture returns a new fixture
func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.k8sObjects = []runtime.Object{}
	f.openebsObjects = []runtime.Object{}
	return f
}

func (f *fixture) SetFakeClient() {
	// Load kubernetes client set by preloading with k8s objects.
	f.k8sClient = fake.NewSimpleClientset(f.k8sObjects...)

	// Load openebs client set by preloading with openebs objects.
	f.openebsClient = openebsFakeClientset.NewSimpleClientset(f.openebsObjects...)
}

func (f *fixture) run_(cvcName string, startInformers bool, expectError bool, testConfig testConfig) {
	c, informers, recorder, err := f.newCVCController()
	if err != nil {
		f.t.Fatalf("error creating cspc controller: %v", err)
	}
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		informers.Start(stopCh)
	}

	defer func(recorder *record.FakeRecorder) {
		close(recorder.Events)
	}(recorder)

	go printEvent(recorder)

	for i := 0; i < testConfig.loopCount; i++ {
		err = c.syncHandler(cvcName)
		if !expectError && err != nil {
			f.t.Errorf("error syncing cvc: %v", err)
		} else if expectError && err == nil {
			f.t.Error("expected error syncing cvc, got nil")
		}
		if testConfig.loopCount > 1 {
			time.Sleep(testConfig.loopDelay)
		}
	}
}

// printEvent prints the events reported by controller
func printEvent(recorder *record.FakeRecorder) {
	rocket := html.UnescapeString("&#128640;")
	warning := html.UnescapeString("&#10071;")
	for {
		msg, ok := <-recorder.Events
		// Channel is closed
		if !ok {
			break
		}
		if strings.Contains(msg, "Normal") {
			// Below line prints 🚀 to identify event
			fmt.Println("Event:  ", rocket, msg)
		} else {
			// Below line prints ❗ to identify event
			fmt.Println("Event:  ", warning, msg)
		}
	}
}

// Returns 0 for resyncPeriod in case resyncing is not needed.
func NoResyncPeriodFunc() time.Duration {
	return 0
}

// newCVCController returns a fake cvc controller
func (f *fixture) newCVCController() (*CVCController, openebsinformers.SharedInformerFactory, *record.FakeRecorder, error) {
	cvcInformerFactory := openebsinformers.NewSharedInformerFactory(f.openebsClient, NoResyncPeriodFunc())

	// Build a fake controller
	c := NewCVCControllerBuilder().
		withOpenEBSClient(f.openebsClient).
		withKubeClient(f.k8sClient).
		withCVCLister(cvcInformerFactory).
		withCVLister(cvcInformerFactory).
		withCVRLister(cvcInformerFactory).
		withEventHandler(cvcInformerFactory).
		withWorkqueueRateLimiting().
		CVCController
	recorder := record.NewFakeRecorder(1024)
	c.recorder = recorder
	c.cvcSynced = alwaysReady
	c.cvrSynced = alwaysReady
	c.enqueueCVCConfig = c.enqueueCVC

	for _, cspi := range f.cspiLister {
		cvcInformerFactory.Cstor().V1().CStorPoolInstances().Informer().GetIndexer().Add(cspi)
	}

	for _, cv := range f.cvLister {
		cvcInformerFactory.Cstor().V1().CStorVolumes().Informer().GetIndexer().Add(cv)
	}
	for _, cvc := range f.cvcLister {
		cvcInformerFactory.Cstor().V1().CStorVolumeConfigs().Informer().GetIndexer().Add(cvc)
	}

	for _, cvr := range f.cvrLister {
		cvcInformerFactory.Cstor().V1().CStorVolumeReplicas().Informer().GetIndexer().Add(cvr)
	}

	return c, cvcInformerFactory, recorder, nil
}

// TestCVCFinalizerRemoval tests the rmoval of cvc protection finalizer
func TestCVCFinalizerRemoval(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	tests := map[string]struct {
		cvc                  *apis.CStorVolumeConfig
		shouldFinalizerExist bool
		testConfig           testConfig
		expectError          bool
	}{
		"When volume deletion triggered": {
			cvc: apis.NewCStorVolumeConfig().
				WithName("cvc-foo-1").
				WithNamespace("openebs").
				WithFinalizer(CStorVolumeConfigFinalizer),
			testConfig: testConfig{
				loopCount: 2,
				loopDelay: time.Second * 1,
			},
			shouldFinalizerExist: false,
			expectError:          false,
		},
	}
	for name, test := range tests {
		name := name
		test := test
		test.cvc.Kind = "CStorVolumeConfig"
		test.cvc.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		t.Run(name, func(t *testing.T) {
			// Create a CVC to persist it in a fake store
			f.openebsClient.CstorV1().CStorVolumeConfigs("openebs").Create(context.TODO(), test.cvc, metav1.CreateOptions{})

			f.run_(testutil.GetKey(test.cvc, t), true, test.expectError, test.testConfig)

			cvc, err := f.openebsClient.CstorV1().CStorVolumeConfigs(test.cvc.Namespace).Get(context.TODO(), test.cvc.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("error getting cvc %s: %v", test.cvc.Name, err)
			}
			if cvc.HasFinalizer(string(CStorVolumeConfigFinalizer)) != test.shouldFinalizerExist {
				t.Errorf(
					"%q test failed %q finalizer exists on %s, expected: %t but got: %t",
					name,
					CStorVolumeConfigFinalizer,
					cvc.Name,
					test.shouldFinalizerExist,
					cvc.HasFinalizer(CStorVolumeConfigFinalizer),
				)
			}
		})
	}
}

func TestVolumeProvisioning(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	f.fakeNodeCreator(5)
	openebsNamespace = "openebs"

	newScheme := runtime.NewScheme()
	newScheme.AddKnownTypes(apis.SchemeGroupVersion, &apis.CStorVolume{})
	scheme.Scheme = newScheme

	tests := map[string]struct {
		cspcName        string
		cvc             *apis.CStorVolumeConfig
		wantCVRCount    int
		wantCVCount     int
		wantTargetCount int
		testConfig      testConfig
	}{

		"One replica provision": {
			cspcName: "cspc-pool1-stripe",
			cvc: &apis.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vol1",
					Namespace: namespace,
					Labels: map[string]string{
						apistypes.CStorPoolClusterLabelKey: "cspc-pool1-stripe",
					},
				},
				Publish: apis.CStorVolumeConfigPublish{
					NodeID: "worker-0",
				},
				Spec: apis.CStorVolumeConfigSpec{
					Capacity: ParseQuantity("5Gi"),
					Provision: apis.VolumeProvision{
						ReplicaCount: 1,
						Capacity:     ParseQuantity("5Gi"),
					},
				},
				Status: apis.CStorVolumeConfigStatus{
					Phase: apis.CStorVolumeConfigPhasePending,
				},
			},
			testConfig: testConfig{
				loopCount: 1,
				loopDelay: time.Second * 1,
			},
			wantCVRCount:    1,
			wantTargetCount: 1,
		},
	}
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			f.fakePoolsCreator(test.cspcName, 2)
			// Create a CVC to persist it in a fake store
			test.cvc.Kind = "CStorVolumeConfig"
			_, err := f.openebsClient.CstorV1().CStorVolumeConfigs("openebs").Create(context.TODO(), test.cvc, metav1.CreateOptions{})
			if err != nil {
				t.Errorf("error creating cvc %s: %v", test.cvc.Name, err)
			}
			f.cvcLister = append(f.cvcLister, test.cvc)

			f.run_(testutil.GetKey(test.cvc, t), true, false, test.testConfig)
			_, err = f.openebsClient.CstorV1().CStorVolumeConfigs(test.cvc.Namespace).Get(context.TODO(), test.cvc.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("error getting cvc %s: %v", test.cvc.Name, err)
			}

			cvrCount := f.getCVRCount(test.cvc.Name, test.cvc.Namespace)
			targetCount := f.getVolumeTargetCount(test.cvc.Name, test.cvc.Namespace)

			if cvrCount != test.wantCVRCount {
				t.Errorf("[Test Case:%s] Want cvr count %d but got %d", name, test.wantCVRCount, cvrCount)
			}

			if targetCount != test.wantTargetCount {
				t.Errorf("[Test Case:%s] Want target count %d but got %d", name, test.wantTargetCount, targetCount)
			}
		})
	}
}

//-------------------------------------older tests----------------------------

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

func ParseQuantity(capacity string) corev1.ResourceList {
	resCapacity, _ := resource.ParseQuantity(capacity)
	resourceList := corev1.ResourceList{
		corev1.ResourceName(corev1.ResourceStorage): resCapacity,
	}
	return resourceList
}

func (f *fixture) getCVRCount(cvcName, cvcNamespace string) int {
	cvrList, err := f.openebsClient.CstorV1().CStorVolumeReplicas(cvcNamespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: "cstorvolume.openebs.io/name" + "=" + cvcName})
	if err != nil {
		f.t.Errorf("failed to list cvrs for cvc %s:%s", cvcName, err)
	}
	return len(cvrList.Items)
}

func (f *fixture) getVolumeTargetCount(cvcName, cvcNamespace string) int {
	deployList, err := f.k8sClient.AppsV1().Deployments(cvcNamespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: "openebs.io/persistent-volume" + "=" + cvcName})
	if err != nil {
		f.t.Errorf("failed to list volume target deployments for cvc %s:%s", cvcName, err)
	}
	return len(deployList.Items)
}

func (f *fixture) fakePoolsCreator(cspcName string, poolCount int) error {
	nodeList, err := f.k8sClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(nodeList.Items) < poolCount {
		return errors.Errorf("enough nodes doesn't exist to create fake CSPIs")
	}
	for i := 0; i < poolCount; i++ {
		labels := map[string]string{
			apistypes.HostNameLabelKey:         nodeList.Items[i].Name,
			apistypes.CStorPoolClusterLabelKey: cspcName,
		}
		cspi := apis.NewCStorPoolInstance().
			WithName(cspcName + "-" + rand.String(4)).
			WithNamespace(namespace).
			WithNodeSelectorByReference(nodeList.Items[i].Labels).
			WithNodeName(nodeList.Items[i].Name).
			WithLabelsNew(labels)
		cspi.Status.Phase = apis.CStorPoolStatusOnline
		_, err := f.openebsClient.CstorV1().CStorPoolInstances(namespace).Create(context.TODO(), cspi, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to create fake cspi")
		}
		//		err = f.createFakePoolPod(cspiObj)
		//	if err != nil {
		//	return errors.Wrapf(err, "failed to create fake pool pod")
		//}
	}
	return nil
}

func (f *fixture) fakeNodeCreator(nodeCount int) {
	for i := 0; i < nodeCount; i++ {
		node := &corev1.Node{}
		node.Name = "worker-" + strconv.Itoa(i)
		labels := make(map[string]string)
		labels["kubernetes.io/hostname"] = node.Name
		node.Labels = labels
		node.Status.Conditions = []corev1.NodeCondition{}
		_, err := f.k8sClient.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
		if err != nil && !k8serror.IsAlreadyExists(err) {
			klog.Error(err)
			continue
		}
		_, err = f.k8sClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
		if err != nil {
			klog.Error(err)
		}
	}
}
