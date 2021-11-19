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
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebscore "github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	"github.com/openebs/api/v3/pkg/apis/types"
	openebsFakeClientset "github.com/openebs/api/v3/pkg/client/clientset/versioned/fake"
	openebsinformers "github.com/openebs/api/v3/pkg/client/informers/externalversions"
	"github.com/openebs/cstor-operators/pkg/controllers/testutil"
	"github.com/openebs/cstor-operators/pkg/cspc/algorithm"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
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
	cspcLister []*cstor.CStorPoolCluster
	cspiLister []*cstor.CStorPoolInstance

	ignoreActionExpectations bool

	// Actions expected to happen on the client. Objects from here are also
	// preloaded into NewSimpleFake.
	actions        []core.Action
	k8sObjects     []runtime.Object
	openebsObjects []runtime.Object
}

func (f *fixture) SetFakeClient() {
	// Load kubernetes client set by preloading with k8s objects.
	f.k8sClient = fake.NewSimpleClientset(f.k8sObjects...)

	// Load openebs client set by preloading with openebs objects.
	f.openebsClient = openebsFakeClientset.NewSimpleClientset(f.openebsObjects...)
}

func (f *fixture) expectUpdateCSPCAction(cspc *cstor.CStorPoolCluster) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "cstorpoolclusters"}, cspc.Namespace, cspc)
	f.actions = append(f.actions, action)
}

func (f *fixture) expectListCSPIAction(cspc *cstor.CStorPoolCluster) {
	action := core.NewListAction(schema.GroupVersionResource{Resource: "cstorpoolinstances"},
		schema.GroupVersionKind{Kind: "cstorpoolinstances"}, cspc.Namespace, metav1.ListOptions{})
	f.actions = append(f.actions, action)
}

func (f *fixture) FakeDiskCreator(totalDisk, totalNode int) {
	// Create some fake block device objects over nodes.
	var key, diskLabel string

	// nodeIdentifer will help in naming a node and attaching multiple disks to a single node.
	nodeIdentifer := 1
	for diskListIndex := 1; diskListIndex <= totalDisk; diskListIndex++ {
		diskIdentifier := strconv.Itoa(diskListIndex)

		if diskListIndex%totalNode == 0 {
			nodeIdentifer++
		}

		key = "ndm.io/blockdevice-type"
		diskLabel = "blockdevice"
		bdObj := &openebscore.BlockDevice{
			TypeMeta: metav1.TypeMeta{
				Kind: "BlockDevices",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "blockdevice-" + diskIdentifier,
				UID:  k8stypes.UID("bdtest" + strconv.Itoa(nodeIdentifer) + diskIdentifier),
				Labels: map[string]string{
					"kubernetes.io/hostname": "worker-" + strconv.Itoa(nodeIdentifer),
					key:                      diskLabel,
				},
			},
			Spec: openebscore.DeviceSpec{
				Details: openebscore.DeviceDetails{
					DeviceType: "disk",
				},
				Partitioned: "NO",
				Capacity: openebscore.DeviceCapacity{
					Storage: 120000000000,
				},
			},
			Status: openebscore.DeviceStatus{
				State: openebscore.BlockDeviceActive,
			},
		}
		_, err := f.openebsClient.OpenebsV1alpha1().BlockDevices("openebs").Create(context.TODO(), bdObj, metav1.CreateOptions{})
		if err != nil {
			klog.Error(err)
		}

	}
}

func (f *fixture) fakeNodeCreator(nodeCount int) {
	// Create 5 nodes

	for i := 0; i < nodeCount; i++ {
		node := &v1.Node{}
		node.Name = "worker-" + strconv.Itoa(i)
		labels := make(map[string]string)
		labels["kubernetes.io/hostname"] = node.Name
		node.Labels = labels
		_, err := f.k8sClient.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
		if err != nil {
			klog.Error(err)
		}
	}
}

// newFixture returns a new fixture
func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.k8sObjects = []runtime.Object{}
	f.openebsObjects = []runtime.Object{}
	//f.k8sClient=fake.NewSimpleClientset()
	//f.openebsClient=openebsFakeClientset.NewSimpleClientset()
	return f
}

func (f *fixture) fakePoolManagerRoutine(signal <-chan bool) {
	for {
		_, ok := <-signal
		// channel is closed
		if !ok {
			break
		}
		poolManagerList, err := f.k8sClient.AppsV1().
			Deployments("openebs").
			List(context.TODO(), metav1.ListOptions{LabelSelector: "app=cstor-pool"})
		if err != nil {
			klog.Infof("Failed to list pool managers error: %v", err)
			continue
		}
		for _, poolManagerObj := range poolManagerList.Items {
			cspiName := poolManagerObj.Name
			_, err := f.openebsClient.CstorV1().CStorPoolInstances("openebs").Get(context.TODO(), cspiName, metav1.GetOptions{})
			if err == nil {
				continue
			}
			if k8serror.IsNotFound(err) {
				err = f.k8sClient.AppsV1().Deployments("openebs").Delete(context.TODO(), cspiName, metav1.DeleteOptions{})
				if err != nil {
					klog.Errorf("failed to delete %s pool manager", cspiName)
					continue
				}
			}
		}
	}
}

func (f *fixture) fakeNDMRoutine() {
	NDMStarted = true
	for {
		bdcList, err := f.openebsClient.OpenebsV1alpha1().BlockDeviceClaims("openebs").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			klog.Error(err)
		}

		bdList, err := f.openebsClient.OpenebsV1alpha1().BlockDevices("openebs").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			klog.Error(err)
		}

		bdNames := make(map[string]string)

		for _, bdc := range bdcList.Items {
			bdNames[bdc.Spec.BlockDeviceName] = bdc.Name
		}

		if err != nil {
			klog.Error(err)
		}

		// Claim the BDs
		for _, bd := range bdList.Items {
			if bdNames[bd.Name] != "" {
				if bd.Status.ClaimState == openebscore.BlockDeviceClaimed {
					continue
				}
				bd.Status.ClaimState = openebscore.BlockDeviceClaimed
				bd.Spec.ClaimRef = &v1.ObjectReference{
					Name: bdNames[bd.Name],
				}
				bd, err := f.openebsClient.OpenebsV1alpha1().BlockDevices("openebs").Update(context.TODO(), &bd, metav1.UpdateOptions{})
				if err != nil {
					klog.Errorf("bd not claimed %s: %s", bd.Name, err.Error())
				}

			}

		}
		time.Sleep(2 * time.Second)
	}
}

// verifyBlockDevicesOfCSPCClaimState will verify status of blockdevices present on CSPC
// if blockdevices were not claimed then it will return error
func (f *fixture) verifyBlockDevicesOfCSPCClaimState(cspc *cstor.CStorPoolCluster) error {
	for _, poolSpec := range cspc.Spec.Pools {
		for _, rg := range append(poolSpec.DataRaidGroups, poolSpec.WriteCacheRaidGroups...) {
			for _, cspiBD := range rg.CStorPoolInstanceBlockDevices {
				bdObj, err := f.openebsClient.OpenebsV1alpha1().
					BlockDevices(cspc.Namespace).
					Get(context.TODO(), cspiBD.BlockDeviceName, metav1.GetOptions{})
				if err != nil {
					return errors.Wrapf(err, "failed to get blcodkevice name %s", bdObj.Name)
				}
				if bdObj.Status.ClaimState != openebscore.BlockDeviceClaimed {
					return errors.Errorf(
						"expected blockdevice %s should be in %s state but got %s",
						bdObj.Name, openebscore.BlockDeviceClaimed, bdObj.Status.ClaimState)
				}
			}
		}
	}
	return nil
}

func (f *fixture) getCSPICount(cspcName, cspcNamespace string) int {
	cspiList, err := f.openebsClient.CstorV1().CStorPoolInstances(cspcNamespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: types.CStorPoolClusterLabelKey + "=" + cspcName})
	if err != nil {
		f.t.Errorf("failed to list cspi for cspc %s:%s", cspcName, err)
	}
	return len(cspiList.Items)
}

func (f *fixture) getPoolManagerCount(cspcName, cspcNamespace string) int {
	deployList, err := f.k8sClient.AppsV1().Deployments(cspcNamespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: types.CStorPoolClusterLabelKey + "=" + cspcName})
	if err != nil {
		f.t.Errorf("failed to list pool manager deployments for cspc %s:%s", cspcName, err)
	}
	return len(deployList.Items)
}

func (f *fixture) run(cspcName string) {
	f.run_(cspcName, true, false, 1, time.Second*0)
}

var NDMStarted bool

func (f *fixture) runLoop(cspcName string, loopCount int, loopDelay time.Duration) {
	go f.fakeNDMRoutine()
	klog.Info("Waiting for fake NDM to start")
	for !NDMStarted {
	}
	f.run_(cspcName, true, false, loopCount, loopDelay)
}

func (f *fixture) run_(cspcName string, startInformers bool, expectError bool, loopCount int, loopDelay time.Duration) {
	c, informers, err := f.newCSPCController()
	if err != nil {
		f.t.Fatalf("error creating cspc controller: %v", err)
	}
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		informers.Start(stopCh)
	}
	notifyPoolManagerRoutine := make(chan bool)
	go f.fakePoolManagerRoutine(notifyPoolManagerRoutine)
	defer close(notifyPoolManagerRoutine)

	// fake ndm go routine
	for i := 0; i < loopCount; i++ {
		err = c.syncCSPC(cspcName)
		if !expectError && err != nil {
			f.t.Errorf("error syncing cspc: %v", err)
		} else if expectError && err == nil {
			f.t.Error("expected error syncing cspc, got nil")
		}
		notifyPoolManagerRoutine <- true

		if !f.ignoreActionExpectations {
			actions := filterInformerActions(f.openebsClient.Actions())
			for i, action := range actions {
				if len(f.actions) < i+1 {
					f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
					break
				}

				expectedAction := f.actions[i]
				if !(expectedAction.Matches(action.GetVerb(), action.GetResource().Resource) && action.GetSubresource() == expectedAction.GetSubresource()) {
					f.t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expectedAction, action)
					continue
				}
			}

			if len(f.actions) > len(actions) {
				f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
			}

		}

		if loopCount > 1 {
			time.Sleep(loopDelay)
		}
	}

}

func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "cstorpoolclusters") ||
				action.Matches("watch", "cstorpoolclusters") ||
				action.Matches("list", "cstorpoolinstances") ||
				action.Matches("watch", "cstorpoolinstances")) {
			continue
		}
		ret = append(ret, action)
	}
	return ret
}

// newCSPCController returns a fake cspc controller
func (f *fixture) newCSPCController() (*Controller, openebsinformers.SharedInformerFactory, error) {
	//// Load kubernetes client set by preloading with k8s objects.
	//f.k8sClient = fake.NewSimpleClientset(f.k8sObjects...)
	//
	//// Load openebs client set by preloading with openebs objects.
	//f.openebsClient = openebsFakeClientset.NewSimpleClientset(f.openebsObjects...)

	cspcInformerFactory := openebsinformers.NewSharedInformerFactory(f.openebsClient, NoResyncPeriodFunc())
	//cspcInformerFactory := informers.NewSharedInformerFactory(openebsClient, getSyncInterval())

	// Build a fake controller
	c := NewControllerBuilder().
		WithOpenEBSClient(f.openebsClient).
		WithKubeClient(f.k8sClient).
		WithCSPCLister(cspcInformerFactory).
		WithCSPILister(cspcInformerFactory).
		WithEventHandler(cspcInformerFactory).
		WithWorkqueueRateLimiting().
		Controller
	c.recorder = &record.FakeRecorder{}
	c.cspcSynced = alwaysReady
	c.cspiSynced = alwaysReady
	c.enqueueCSPC = c.enqueue

	for _, d := range f.cspcLister {
		cspcInformerFactory.Cstor().V1().CStorPoolClusters().Informer().GetIndexer().Add(d)
	}

	for _, rs := range f.cspiLister {
		cspcInformerFactory.Cstor().V1().CStorPoolInstances().Informer().GetIndexer().Add(rs)
	}

	return c, cspcInformerFactory, nil
}

// NoProvisionExpectations are the actions that are surely carried out when a brand new cspc enter into
// the etcd and has no pool specs to provision cstor pools on nodes.
func (f *fixture) NoProvisionExpectations(cspc *cstor.CStorPoolCluster) {
	// Following expectations are due to addition of version and finalizers on cspc and cleanup that might be required.
	// ToDO: Improve on cspi listing by using cspi lister but this will require some thought
	// as there could be stale cspis reported by the lister which actually is not present in the system.
	// These are the actions that are surely carried out when a brand new cspc enter into the etcd and has no
	// pool specs to provision cstor pools on nodes.
	f.expectListCSPIAction(cspc)
	f.expectListCSPIAction(cspc)
	f.expectListCSPIAction(cspc)
	f.expectUpdateCSPCAction(cspc)
	f.expectUpdateCSPCAction(cspc)
	f.expectUpdateCSPCAction(cspc)
	f.expectListCSPIAction(cspc)
	f.expectListCSPIAction(cspc)
	f.expectListCSPIAction(cspc)
	f.expectUpdateCSPCAction(cspc)
}

//-------------------------------------------*Non-Provisioning Tests*---------------------------------------------------

// TestCSPCFinalizerAdd tests the addition of cspc finalizer when a brand new cspc is created
func TestCSPCFinalizerAdd(t *testing.T) {
	f := newFixture(t)
	cspc := cstor.NewCStorPoolCluster().
		WithName("cspc-foo").
		WithNamespace("openebs")
	cspc.Kind = "CStorPoolCluster"

	f.cspcLister = append(f.cspcLister, cspc)
	f.openebsObjects = append(f.openebsObjects, cspc)
	f.SetFakeClient()
	f.NoProvisionExpectations(cspc)

	f.run(testutil.GetKey(cspc, t))

	cspc, err := f.openebsClient.CstorV1().CStorPoolClusters(cspc.Namespace).Get(context.TODO(), cspc.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("error getting cspc %s: %v", cspc.Name, err)
	}

	if !cspc.HasFinalizer(types.CSPCFinalizer) {
		t.Errorf("expected finalizer %s on cspc %s but was not found:", types.CSPCFinalizer, cspc.Name)
	}
}

//-----------------------------------------------*Provisioning Tests*---------------------------------------------------

// TestCSPCProvisionSingleNode tests the provisioning of cstor pool on single node.
func TestCSPCProvisionSingleNode(t *testing.T) {
	fixture := newFixture(t)
	fixture.SetFakeClient()
	fixture.FakeDiskCreator(70, 5)
	fixture.fakeNodeCreator(5)

	tests := map[string]struct {
		CSPC                       *cstor.CStorPoolCluster
		wantCSPICount              int
		wantPoolManagerCount       int
		wantBlockDeviceCountInCSPI int
	}{
		"One node stripe pool provision": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("stripe")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
							WithName("blockdevice-1")))),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 1,
		},

		"One node mirror pool provision": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("mirror")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
							WithName("blockdevice-2"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-3")))),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 2,
		},

		"One node raidz pool provision": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-0"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("raidz")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
							WithName("blockdevice-4"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-6"),
						))),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 3,
		},

		"One node raidz2 pool provision": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz2").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("raidz2")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-7"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-8"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-9"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-10"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-11"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-12"),
						))),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 6,
		},
		"One node stripe pool provision with writecache raidgroup": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe-writecache").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("stripe").
						WithWriteCacheGroupType("stripe")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
							WithName("blockdevice-13"))).
					WithWriteCacheRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-14"),
						))),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 1,
		},
		"One node mirror pool provision with multiple data and writecache raidgroup": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror-writecache").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("mirror").
						WithWriteCacheGroupType("mirrror")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
							WithName("blockdevice-15"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-16")),
						*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
								WithName("blockdevice-17"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-18"))).
					WithWriteCacheRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-19"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-20"),
						).
						WithCStorPoolInstanceBlockDevices(
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-21"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-22"),
						),
					),
				),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 2,
		},
	}

	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			test.CSPC.Kind = "CStorPoolCluster"
			// Create a CSPC to persist it in a fake store
			fixture.openebsClient.CstorV1().CStorPoolClusters("openebs").Create(context.TODO(), test.CSPC, metav1.CreateOptions{})
			// Add the cspc to the cspc lister
			fixture.cspcLister = append(fixture.cspcLister, test.CSPC)
			// We do not want to track the API calls here for provisioning rather the state of the system
			// hence ignore the action expectations.
			// Although a diff test aiming to track/benchmark API calls for diff paths of cspc controller
			// should be in a different test(todo).
			fixture.ignoreActionExpectations = true
			fixture.runLoop(testutil.GetKey(test.CSPC, t), 10, time.Second*1)
			gotCSPICount := fixture.getCSPICount(test.CSPC.Name, test.CSPC.Namespace)
			gotPoolManagerCount := fixture.getPoolManagerCount(test.CSPC.Name, test.CSPC.Namespace)

			if gotCSPICount != test.wantCSPICount {
				t.Errorf("[Test Case:%s] Want cspi count %d but got %d", name, test.wantCSPICount, gotCSPICount)
			}

			if gotPoolManagerCount != test.wantPoolManagerCount {
				t.Errorf("[Test Case:%s] Want pool manager count %d but got %d",
					name, test.wantPoolManagerCount, gotPoolManagerCount)

			}
			cspiList, err := fixture.openebsClient.CstorV1().
				CStorPoolInstances("openebs").
				List(context.TODO(), metav1.ListOptions{LabelSelector: types.CStorPoolClusterLabelKey + "=" + test.CSPC.Name})

			if err != nil {
				t.Errorf("[Test Case:%s] fake client failed to list cspi for cspc %s:%s", name, test.CSPC.Name, err)
			}

			for _, cspi := range cspiList.Items {
				bdCount := len(cspi.Spec.DataRaidGroups[0].CStorPoolInstanceBlockDevices)
				if bdCount != test.wantBlockDeviceCountInCSPI {
					t.Errorf("[Test Case:%s] want bd count %d but"+
						" got %d for cspi %s", name, test.wantBlockDeviceCountInCSPI, bdCount, cspi.Name)
				}
			}

			err = fixture.verifyBlockDevicesOfCSPCClaimState(test.CSPC)
			if err != nil {
				t.Errorf("[Test Case: %s] failed error: %v", name, err)
			}
		})
	}
}

// TestCSPCProvisionMultipleNode tests the provisioning of cstor pool on multiple node with multiple raid groups.
func TestCSPCProvisionMultipleNode(t *testing.T) {
	fixture := newFixture(t)
	fixture.SetFakeClient()
	fixture.FakeDiskCreator(150, 5)
	fixture.fakeNodeCreator(5)

	tests := map[string]struct {
		CSPC                       *cstor.CStorPoolCluster
		wantCSPICount              int
		wantPoolManagerCount       int
		wantBlockDeviceCountInCSPI int
		wantRaidGroupCountInCSPI   int
	}{
		"3 node stripe with 1 raid group pool provision": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-1"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-2"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-31"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-32"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-3"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-61"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-62"),
								),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 2,
			wantRaidGroupCountInCSPI:   1,
		},

		"3 node mirror pool with 2 raid group provision": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-3"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-4")),

							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-6")),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-33"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-34")),

							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-35"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-36")),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-3"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-63"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-64")),

							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-65"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-66")),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 2,
			wantRaidGroupCountInCSPI:   2,
		},

		"3 node raidz pool with 2 raid groups provision": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(
							*cstor.NewPoolConfig().WithDataRaidGroupType("raidz")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-7"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-8"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-9"),
							),
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-10"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-11"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-12"),
							),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(
							*cstor.NewPoolConfig().WithDataRaidGroupType("raidz")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-37"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-38"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-39"),
							),
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-40"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-41"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-42"),
							),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-3"}).
						WithPoolConfig(
							*cstor.NewPoolConfig().WithDataRaidGroupType("raidz")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-67"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-68"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-69"),
							),
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-70"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-71"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-72"),
							),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 3,
			wantRaidGroupCountInCSPI:   2,
		},

		"3 node raidz2 pool provision": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz2").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("raidz2")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-13"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-14"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-15"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-16"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-17"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-18"),
							),
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-19"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-20"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-21"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-22"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-23"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-24"),
							),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("raidz2")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-43"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-44"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-45"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-46"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-47"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-48"),
							),

							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-49"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-50"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-51"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-52"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-53"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-54"),
							),
						),
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-3"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("raidz2")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-73"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-74"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-75"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-76"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-77"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-78"),
							),

							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-79"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-80"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-81"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-82"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-83"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-84"),
							),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 6,
			wantRaidGroupCountInCSPI:   2,
		},
		"3 node with combination of different raid groups and provision pool": {
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-combinations").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe").
							WithWriteCacheGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-25"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-26"),
								),
						).
						WithWriteCacheRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-27"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-28"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror").
							WithWriteCacheGroupType("raidz")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-55"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-56"),
								),
						).
						WithWriteCacheRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-57"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-58"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-59"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-3"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe").
							WithWriteCacheGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-85"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-86"),
								),
						).
						WithWriteCacheRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-87"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-88"),
								),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 2,
			wantRaidGroupCountInCSPI:   1,
		},
	}

	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			test.CSPC.Kind = "CStorPoolCluster"
			// Create a CSPC to persist it in a fake store
			fixture.openebsClient.CstorV1().CStorPoolClusters("openebs").Create(context.TODO(), test.CSPC, metav1.CreateOptions{})
			// Add the cspc to the cspc lister
			fixture.cspcLister = append(fixture.cspcLister, test.CSPC)
			// We do not want to track the API calls here for provisioning rather the state of the system
			// hence ignore the action expectations.
			// Although a diff test aiming to track/benchmark API calls for diff paths of cspc controller
			// should be in a different test(todo).
			fixture.ignoreActionExpectations = true
			fixture.runLoop(testutil.GetKey(test.CSPC, t), 10, time.Second*1)
			gotCSPICount := fixture.getCSPICount(test.CSPC.Name, test.CSPC.Namespace)
			gotPoolManagerCount := fixture.getPoolManagerCount(test.CSPC.Name, test.CSPC.Namespace)

			if gotCSPICount != test.wantCSPICount {
				t.Errorf("[Test Case:%s] Want cspi count %d but got %d", name, test.wantCSPICount, gotCSPICount)
			}

			if gotPoolManagerCount != test.wantPoolManagerCount {
				t.Errorf("[Test Case:%s] Want pool manager count %d but got %d",
					name, test.wantPoolManagerCount, gotPoolManagerCount)

			}
			cspiList, err := fixture.openebsClient.CstorV1().
				CStorPoolInstances("openebs").
				List(context.TODO(), metav1.ListOptions{LabelSelector: types.CStorPoolClusterLabelKey + "=" + test.CSPC.Name})

			if err != nil {
				t.Errorf("[Test Case:%s] fake client failed to list cspi for cspc %s:%s", name, test.CSPC.Name, err)
			}

			for _, cspi := range cspiList.Items {
				dataRaidGroups := cspi.Spec.DataRaidGroups
				gotRaidGroupCount := len(dataRaidGroups)
				if gotRaidGroupCount != test.wantRaidGroupCountInCSPI {
					t.Errorf("[Test Case:%s] want raid group count %d but"+
						" got %d for cspi %s", name, test.wantRaidGroupCountInCSPI, gotRaidGroupCount, cspi.Name)
				}
				for _, rg := range dataRaidGroups {
					bdCount := len(rg.CStorPoolInstanceBlockDevices)
					if bdCount != test.wantBlockDeviceCountInCSPI {
						t.Errorf("[Test Case:%s] want bd count %d but"+
							" got %d for cspi %s", name, test.wantBlockDeviceCountInCSPI, bdCount, cspi.Name)
					}
				}

			}
			err = fixture.verifyBlockDevicesOfCSPCClaimState(test.CSPC)
			if err != nil {
				t.Errorf("[Test Case: %s] failed error: %v", name, err)
			}
		})
	}
}

//-------------------------------------------*Day-2 Operations Tests*---------------------------------------------------

// TestPoolScaleUp tests for PoolScaleUP -- Horizontal Scaling.
func TestPoolScaleUp(t *testing.T) {
	fixture := newFixture(t)
	fixture.SetFakeClient()
	fixture.FakeDiskCreator(70, 5)
	fixture.fakeNodeCreator(5)

	tests := []struct {
		TestName                   string
		CSPCApply                  bool
		CSPC                       *cstor.CStorPoolCluster
		wantCSPICount              int
		wantPoolManagerCount       int
		wantBlockDeviceCountInCSPI int
	}{
		{
			TestName:  "[Pre-requisite Step for Pool Scale Up] Provision a 1 node stripe pool",
			CSPCApply: false,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("stripe")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
							WithName("blockdevice-1")))),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 1,
		},

		{
			TestName:  "Scale up stripe pool to 3 pools",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
								WithName("blockdevice-1"))),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-31"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-3"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-61"),
								),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.TestName, func(t *testing.T) {
			test.CSPC.Kind = "CStorPoolCluster"
			// Create a CSPC to persist it in a fake store
			if test.CSPCApply {
				_, err := fixture.openebsClient.CstorV1().CStorPoolClusters("openebs").Update(context.TODO(), test.CSPC, metav1.UpdateOptions{})
				if err != nil {
					t.Errorf("[Test Case:%s] failed to update cspc %s", test.TestName, test.CSPC.Name)

				}
			} else {
				_, err := fixture.openebsClient.CstorV1().CStorPoolClusters("openebs").Create(context.TODO(), test.CSPC, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("[Test Case:%s] failed to create cspc %s", test.TestName, test.CSPC.Name)

				}
			}

			// Add the cspc to the cspc lister
			fixture.cspcLister = append(fixture.cspcLister, test.CSPC)
			// We do not want to track the API calls here for provisioning rather the state of the system
			// hence ignore the action expectations.
			// Although a diff test aiming to track/benchmark API calls for diff paths of cspc controller
			// should be in a different test(todo).
			fixture.ignoreActionExpectations = true
			fixture.runLoop(testutil.GetKey(test.CSPC, t), 10, time.Second*1)
			gotCSPICount := fixture.getCSPICount(test.CSPC.Name, test.CSPC.Namespace)
			gotPoolManagerCount := fixture.getPoolManagerCount(test.CSPC.Name, test.CSPC.Namespace)

			if gotCSPICount != test.wantCSPICount {
				t.Errorf("[Test Case:%s] Want cspi count %d but got %d", test.TestName, test.wantCSPICount, gotCSPICount)
			}

			if gotPoolManagerCount != test.wantPoolManagerCount {
				t.Errorf("[Test Case:%s] Want pool manager count %d but got %d",
					test.TestName, test.wantPoolManagerCount, gotPoolManagerCount)

			}
			cspiList, err := fixture.openebsClient.CstorV1().
				CStorPoolInstances("openebs").
				List(context.TODO(), metav1.ListOptions{LabelSelector: types.CStorPoolClusterLabelKey + "=" + test.CSPC.Name})

			if err != nil {
				t.Errorf("[Test Case:%s] fake client failed to list cspi for cspc %s:%s", test.TestName, test.CSPC.Name, err)
			}

			for _, cspi := range cspiList.Items {
				bdCount := len(cspi.Spec.DataRaidGroups[0].CStorPoolInstanceBlockDevices)
				if bdCount != test.wantBlockDeviceCountInCSPI {
					t.Errorf("[Test Case:%s] want bd count %d but"+
						" got %d for cspi %s", test.TestName, test.wantBlockDeviceCountInCSPI, bdCount, cspi.Name)
				}
			}

			err = fixture.verifyBlockDevicesOfCSPCClaimState(test.CSPC)
			if err != nil {
				t.Errorf("[Test Case: %s] failed error: %v", test.TestName, err)
			}
		})
	}
}

// TestPoolExpansion tests for Pool expansion -- Vertical Scaling.
// Note: Vertical expansion here means -- expansion of existing pool when a new blockdevice is added.
// There is also a case of expansion when the storage size of a virtual disk expands.
// This function does not test for expansion when the storage size of a virtual disk expands.
func TestPoolPoolExpansion(t *testing.T) {
	fixture := newFixture(t)
	fixture.SetFakeClient()
	fixture.FakeDiskCreator(70, 5)
	fixture.fakeNodeCreator(5)

	tests := []struct {
		TestName                   string
		CSPCApply                  bool
		CSPC                       *cstor.CStorPoolCluster
		wantCSPICount              int
		wantPoolManagerCount       int
		wantBlockDeviceCountInCSPI int
		wantRaidGroupCount         int
	}{
		{
			TestName: "[Pre-requisite Step for Pool expansion] " +
				"Provision a 1 node stripe pool with 1 blockdevice",
			CSPCApply: false,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("stripe")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
							WithName("blockdevice-1")))),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 1,
			wantRaidGroupCount:         1,
		},

		{
			TestName:  "Add 1 more block device to expand the stripe pool",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-1"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-2"),
							)),
				),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 2,
			wantRaidGroupCount:         1,
		},
		{
			TestName: "[Pre-requisite Step for Pool expansion] " +
				"Provision a 1 node mirror pool with 2 blockdevices",
			CSPCApply: false,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("mirror")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
							WithName("blockdevice-3"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-4")))),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 2,
			wantRaidGroupCount:         1,
		},
		{
			TestName:  "Add 1 more raid group to expand the mirror pool",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("mirror")).
					WithDataRaidGroups(
						*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-3"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-4")),

						*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-6")),
					)),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 2,
			wantRaidGroupCount:         2,
		},

		{
			TestName: "[Pre-requisite Step for Pool expansion] " +
				"Provision a 1 node raidz pool with 3 blockdevices",
			CSPCApply: false,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("raidz")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-7"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-8"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-9"),
						))),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 3,
			wantRaidGroupCount:         1,
		},
		{
			TestName:  "Add 1 more raid group to expand the raidz pool",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("mirror")).
					WithDataRaidGroups(
						*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-7"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-8"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-9"),
							),

						*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-10"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-11"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-12"),
							),
					)),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 3,
			wantRaidGroupCount:         2,
		},

		{
			TestName: "[Pre-requisite Step for Pool expansion] " +
				"Provision a 1 node raidz2 pool with 6 blockdevices",
			CSPCApply: false,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz2").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("raidz2")).
					WithDataRaidGroups(*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-21"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-22"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-23"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-24"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-25"),
							*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-26"),
						))),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 6,
			wantRaidGroupCount:         1,
		},
		{
			TestName:  "Add 1 more raid group to expand the raidz pool",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz2").
				WithNamespace("openebs").
				WithPoolSpecs(*cstor.NewPoolSpec().
					WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
					WithPoolConfig(*cstor.NewPoolConfig().
						WithDataRaidGroupType("raidz2")).
					WithDataRaidGroups(
						*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-21"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-22"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-23"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-24"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-25"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-26"),
							),

						*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-27"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-28"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-29"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-30"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-31"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-32"),
							),
					)),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 6,
			wantRaidGroupCount:         2,
		},
		{
			TestName:  "Add 1 write cache raid group block devices",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-1"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-2"),
							)).
						WithWriteCacheRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-3"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-4"),
							)),
				),
			wantCSPICount:              1,
			wantPoolManagerCount:       1,
			wantBlockDeviceCountInCSPI: 2,
			wantRaidGroupCount:         1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.TestName, func(t *testing.T) {
			test.CSPC.Kind = "CStorPoolCluster"
			// Create a CSPC to persist it in a fake store
			var gotCSPC *cstor.CStorPoolCluster
			var errCSPC error
			if test.CSPCApply {
				gotCSPC, errCSPC = fixture.openebsClient.CstorV1().CStorPoolClusters("openebs").Update(context.TODO(), test.CSPC, metav1.UpdateOptions{})
				if errCSPC != nil {
					t.Errorf("[Test Case:%s] failed to update cspc %s:%s", test.TestName, test.CSPC.Name, errCSPC)

				}
			} else {
				gotCSPC, errCSPC = fixture.openebsClient.CstorV1().CStorPoolClusters("openebs").Create(context.TODO(), test.CSPC, metav1.CreateOptions{})
				if errCSPC != nil {
					t.Errorf("[Test Case:%s] failed to create cspc %s:%s", test.TestName, test.CSPC.Name, errCSPC)

				}
			}

			// Add the cspc to the cspc lister
			fixture.cspcLister = append(fixture.cspcLister, gotCSPC)
			// We do not want to track the API calls here for provisioning rather the state of the system
			// hence ignore the action expectations.
			// Although a diff test aiming to track/benchmark API calls for diff paths of cspc controller
			// should be in a different test(todo).
			fixture.ignoreActionExpectations = true
			fixture.runLoop(testutil.GetKey(test.CSPC, t), 10, time.Second*1)
			gotCSPICount := fixture.getCSPICount(test.CSPC.Name, test.CSPC.Namespace)
			gotPoolManagerCount := fixture.getPoolManagerCount(test.CSPC.Name, test.CSPC.Namespace)

			if gotCSPICount != test.wantCSPICount {
				t.Errorf("[Test Case:%s] Want cspi count %d but got %d", test.TestName, test.wantCSPICount, gotCSPICount)
			}

			if gotPoolManagerCount != test.wantPoolManagerCount {
				t.Errorf("[Test Case:%s] Want pool manager count %d but got %d",
					test.TestName, test.wantPoolManagerCount, gotPoolManagerCount)

			}
			cspiList, err := fixture.openebsClient.CstorV1().
				CStorPoolInstances("openebs").
				List(context.TODO(), metav1.ListOptions{LabelSelector: types.CStorPoolClusterLabelKey + "=" + test.CSPC.Name})

			if err != nil {
				t.Errorf("[Test Case:%s] fake client failed to list cspi for cspc %s:%s", test.TestName, test.CSPC.Name, err)
			}

			for _, cspi := range cspiList.Items {
				bdCount := len(cspi.Spec.DataRaidGroups[0].CStorPoolInstanceBlockDevices)
				if bdCount != test.wantBlockDeviceCountInCSPI {
					t.Errorf("[Test Case:%s] want bd count %d but"+
						" got %d for cspi %s", test.TestName, test.wantBlockDeviceCountInCSPI, bdCount, cspi.Name)
				}
			}
			err = fixture.verifyBlockDevicesOfCSPCClaimState(test.CSPC)
			if err != nil {
				t.Errorf("[Test Case: %s] failed error: %v", test.TestName, err)
			}
		})
	}
}

// TestCSPCNodeNameChanges after provisioning the cstor pools.
// After provisioning the pool if node name changes CSPC
// controller should shift the pool-mangers to different node
func TestCSPCNodeNameChanges(t *testing.T) {
	fixture := newFixture(t)
	fixture.SetFakeClient()
	fixture.FakeDiskCreator(150, 6)
	fixture.fakeNodeCreator(6)

	tests := []struct {
		testName                   string
		CSPC                       *cstor.CStorPoolCluster
		CSPCApply                  bool
		replacedBlockDeviceList    []string
		wantCSPICount              int
		wantPoolManagerCount       int
		wantBlockDeviceCountInCSPI int
	}{
		{
			testName: "[Pre-requisite Step for node name change in stripe multiple node] " +
				"3 node stripe with 1 raid group pool provision",
			CSPCApply: false,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-1"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-2"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-31"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-32"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-3"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-61"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-62"),
								),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 2,
		},
		{
			testName:  "Changed worker-1 and worker-3 to worker-5 and worker-4",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-5"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-1"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-2"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-31"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-32"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-4"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-61"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-62"),
								),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 2,
		},
		{
			testName:  "Changed worker-5 and worker-4 to worker-1 and worker-3 and expanded the pool",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-1"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-2"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-3"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-31"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-32"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-33"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-3"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-61"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-62"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-63"),
								),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 3,
		},
		{
			testName:  "Changed worker-2 to worker-5 and deleted the pool spec from worker-3",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-1"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-2"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-3"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-5"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-31"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-32"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-33"),
								),
						),
				),
			wantCSPICount:              2,
			wantPoolManagerCount:       2,
			wantBlockDeviceCountInCSPI: 3,
		},
		{
			testName: "[Pre-requisite Step for node name change in mirror node] " +
				"3 node stripe with 1 raid group pool provision",
			CSPCApply: false,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-4"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-34"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-35"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-3"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-64"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-65"),
								),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 2,
		},
		{
			testName:  "worker-2 and worker-3 name changed to worker-5 and worker-4 along with blockdevice replacement",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-4"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-5"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-35"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-36"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-4"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-66"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-65"),
								),
						),
				),
			replacedBlockDeviceList:    []string{"blockdevice-36", "blockdevice-66"},
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 2,
		},
		{
			testName:  "worker-5 changed to worker-2 and by expanding the pool vertically and scaling down the pools vertically",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-4"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-37"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-38"),
								),
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-35"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-36"),
								),
						),
				),
			wantCSPICount:              2,
			wantPoolManagerCount:       2,
			wantBlockDeviceCountInCSPI: 2,
		},
		{
			testName:  "worker-1 changed to worker-5 and by expanding the pool horizontally",
			CSPCApply: true,
			CSPC: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-5"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-4"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-37"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-38"),
								),
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-35"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-36"),
								),
						),

					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-3"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().
								WithCStorPoolInstanceBlockDevices(
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-90"),
									*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-91"),
								),
						),
				),
			wantCSPICount:              3,
			wantPoolManagerCount:       3,
			wantBlockDeviceCountInCSPI: 2,
		},
	}

	for _, test := range tests {
		test := test
		test.CSPC.Kind = "CStorPoolCluster"
		// Create a CSPC to persist it in a fake store
		var gotCSPC *cstor.CStorPoolCluster
		var errCSPC error
		hostNamesMap := map[string]int{}
		if test.CSPCApply {
			gotCSPC, errCSPC = fixture.openebsClient.CstorV1().CStorPoolClusters("openebs").Update(context.TODO(), test.CSPC, metav1.UpdateOptions{})
			if errCSPC != nil {
				t.Errorf("[Test Case:%s] failed to update cspc %s:%s", test.testName, test.CSPC.Name, errCSPC)

			}
		} else {
			gotCSPC, errCSPC = fixture.openebsClient.CstorV1().CStorPoolClusters("openebs").Create(context.TODO(), test.CSPC, metav1.CreateOptions{})
			if errCSPC != nil {
				t.Errorf("[Test Case:%s] failed to create cspc %s:%s", test.testName, test.CSPC.Name, errCSPC)
			}
		}
		ac, _ := algorithm.NewBuilder().
			WithCSPC(gotCSPC).
			WithNameSpace(gotCSPC.Namespace).
			WithKubeClient(fixture.k8sClient).
			WithOpenEBSClient(fixture.openebsClient).
			Build()
		for _, bdName := range test.replacedBlockDeviceList {
			bdObj, err := fixture.openebsClient.
				OpenebsV1alpha1().
				BlockDevices("openebs").
				Get(context.TODO(), bdName, metav1.GetOptions{})
			if err != nil {
				t.Errorf("[Test Case: %s] failed to get the blockdevice error: %v", test.testName, err)
			}
			err = ac.ClaimBD(*bdObj)
			if err != nil {
				t.Errorf("[Test Case: %s] failed to claim blockdevices error: %v", test.testName, err)
			}
		}
		for _, poolSpec := range gotCSPC.Spec.Pools {
			hostNamesMap[poolSpec.NodeSelector[types.HostNameLabelKey]]++
		}
		// Add the cspc to the cspc lister
		fixture.cspcLister = append(fixture.cspcLister, gotCSPC)
		// We do not want to track the API calls here for provisioning rather the state of the system
		// hence ignore the action expectations.
		// Although a diff test aiming to track/benchmark API calls for diff paths of cspc controller
		// should be in a different test(todo).
		fixture.ignoreActionExpectations = true

		fixture.runLoop(testutil.GetKey(gotCSPC, t), 5, time.Second*1)
		gotCSPICount := fixture.getCSPICount(gotCSPC.Name, gotCSPC.Namespace)
		gotPoolManagerCount := fixture.getPoolManagerCount(gotCSPC.Name, gotCSPC.Namespace)

		if gotCSPICount != test.wantCSPICount {
			t.Errorf("[Test Case:%s] Want cspi count %d but got %d", test.testName, test.wantCSPICount, gotCSPICount)
		}

		if gotPoolManagerCount != test.wantPoolManagerCount {
			t.Errorf("[Test Case:%s] Want pool manager count %d but got %d",
				test.testName, test.wantPoolManagerCount, gotPoolManagerCount)
		}

		cspiList, err := fixture.openebsClient.CstorV1().
			CStorPoolInstances("openebs").
			List(context.TODO(), metav1.ListOptions{LabelSelector: types.CStorPoolClusterLabelKey + "=" + gotCSPC.Name})
		if err != nil {
			t.Errorf("[Test Case:%s] fake client failed to list cspi for cspc %s:%s", test.testName, gotCSPC.Name, err)
		}

		// Verify node name changes
		for _, cspi := range cspiList.Items {
			hostNamesMap[cspi.Spec.HostName]++
			bdCount := len(cspi.Spec.DataRaidGroups[0].CStorPoolInstanceBlockDevices)

			foundPoolSpec := false
			for _, cspcPoolSpec := range gotCSPC.Spec.Pools {
				if reflect.DeepEqual(cspcPoolSpec.NodeSelector, cspi.Spec.NodeSelector) {
					foundPoolSpec = true
					break
				}
			}
			if !foundPoolSpec {
				t.Errorf("failed to find the pool spec belongs to %s", cspi.Name)
			}

			if bdCount != test.wantBlockDeviceCountInCSPI {
				t.Errorf("[Test Case:%s] want bd count %d but"+
					" got %d for cspi %s", test.testName, test.wantBlockDeviceCountInCSPI, bdCount, cspi.Name)
			}

			poolManager, err := fixture.k8sClient.
				AppsV1().
				Deployments(test.CSPC.Namespace).
				Get(context.TODO(), cspi.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("failed to get pool manager %s, err: %v", cspi.Name, err)
			}
			cspiHostName := cspi.Spec.HostName
			poolManagerHostName := poolManager.Spec.Template.Spec.NodeSelector[types.HostNameLabelKey]
			if poolManagerHostName != cspiHostName {
				t.Errorf("[Test Case:%s] want pool managet to be on host %s but"+
					" got %s", test.testName, cspiHostName, poolManagerHostName)
			}
		}
		for key, value := range hostNamesMap {
			if value != 2 {
				t.Errorf("[Test Case: %s] CSPI might/might not exist on host %s -- value %d", test.testName, key, value)
			}
		}
	}
}
