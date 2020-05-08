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
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	openebscore "github.com/openebs/api/pkg/apis/openebs.io/v1alpha1"
	"github.com/openebs/api/pkg/apis/types"
	openebsFakeClientset "github.com/openebs/api/pkg/client/clientset/versioned/fake"
	openebsinformers "github.com/openebs/api/pkg/client/informers/externalversions"
	"github.com/openebs/api/pkg/util"
	"github.com/openebs/cstor-operators/pkg/controllers/common"
	"github.com/openebs/cstor-operators/pkg/controllers/testutil"
	executor "github.com/openebs/cstor-operators/pkg/controllers/testutil/zcmd/executor"
	zpool "github.com/openebs/cstor-operators/pkg/controllers/testutil/zcmd/zpool"
	"github.com/openebs/cstor-operators/pkg/version"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

var (
	alwaysReady = func() bool { return true }
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
	cspiLister []*cstor.CStorPoolInstance

	ignoreActionExpectations bool

	// Actions expected to happen on the client. Objects from here are also
	// preloaded into NewSimpleFake.
	actions        []core.Action
	k8sObjects     []runtime.Object
	openebsObjects []runtime.Object
}

// testConfig contains the information required to run the test
type testConfig struct {
	// To Add Data RaidGroups in cStor pool
	dataRaidGroups []cstor.RaidGroup
	// To Add WriteCache RaidGroups in cStor pool
	writeCacheRaidGroups []cstor.RaidGroup
	// writeCacheGroupType defines the writecache raid group
	writeCacheGroupType string
	// ReplaceBlockDevices in cStor pool
	replaceBlcokDevices map[string]string
	// isDay2OperationNeedToPerform is set then above operations will be performed
	isDay2OperationNeedToPerform bool
	// loopCount times reconcile function will be called
	loopCount int
	// time interval to trigger reconciliation
	loopDelay time.Duration
	// poolInfo will usefull to execute pool commands
	poolInfo *zpool.MockPoolInfo
}

// newFixture returns a new fixture
func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.k8sObjects = []runtime.Object{}
	f.openebsObjects = []runtime.Object{}
	return f
}

// SetFakeClient initilizes the fake required clientsets
func (f *fixture) SetFakeClient() {
	// Load kubernetes client set by preloading with k8s objects.
	f.k8sClient = fake.NewSimpleClientset(f.k8sObjects...)

	// Load openebs client set by preloading with openebs objects.
	f.openebsClient = openebsFakeClientset.NewSimpleClientset(f.openebsObjects...)
}

// Returns 0 for resyncPeriod in case resyncing is not needed.
func NoResyncPeriodFunc() time.Duration {
	return 0
}

// newCSPIController returns a fake cspi controller
func (f *fixture) newCSPIController(
	poolInfo *zpool.MockPoolInfo) (*CStorPoolInstanceController, openebsinformers.SharedInformerFactory, *record.FakeRecorder, error) {
	//// Load kubernetes client set by preloading with k8s objects.
	//f.k8sClient = fake.NewSimpleClientset(f.k8sObjects...)
	//
	//// Load openebs client set by preloading with openebs objects.
	//f.openebsClient = openebsFakeClientset.NewSimpleClientset(f.openebsObjects...)

	fakeZCMDExecutor := executor.NewFakeZCommandFromPoolInfo(poolInfo)

	cspiInformerFactory := openebsinformers.NewSharedInformerFactory(f.openebsClient, NoResyncPeriodFunc())
	//cspiInformerFactory := informers.NewSharedInformerFactory(openebsClient, getSyncInterval())

	recorder := record.NewFakeRecorder(1024)
	// Build a fake controller
	controller := &CStorPoolInstanceController{
		kubeclientset:           f.k8sClient,
		clientset:               f.openebsClient,
		cStorPoolInstanceSynced: alwaysReady,
		workqueue:               workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), poolControllerName),
		recorder:                recorder,
		zcmdExecutor:            fakeZCMDExecutor,
	}

	for _, rs := range f.cspiLister {
		cspiInformerFactory.Cstor().V1().CStorPoolInstances().Informer().GetIndexer().Add(rs)
	}

	// returning recorder to print cspi controller events
	return controller, cspiInformerFactory, recorder, nil
}

// CreateFakeBlockDevices creates the fake blockdevices
// created blockdevice
// NOTE: It will create all blockdevices on single node
func (f *fixture) createFakeBlockDevices(totalDisk int, hostName string) {
	// Create some fake block device objects over nodes.
	var key, diskLabel string

	for diskListIndex := 1; diskListIndex <= totalDisk; diskListIndex++ {
		diskIdentifier := strconv.Itoa(diskListIndex)

		path1 := fmt.Sprintf("/dev/disk/by-id/ata-WDC_WD10JPVX-%s-WXG1A-%s", rand.String(4), diskIdentifier)
		path2 := fmt.Sprintf("/dev/disk/by-path/pci-0000:00:%s.0-ata-1", diskIdentifier)
		key = "ndm.io/blockdevice-type"
		diskLabel = "blockdevice"
		bdObj := &openebscore.BlockDevice{
			TypeMeta: metav1.TypeMeta{
				Kind: "BlockDevices",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "blockdevice-" + diskIdentifier,
				UID:  k8stypes.UID("bdtest" + hostName + diskIdentifier),
				Labels: map[string]string{
					"kubernetes.io/hostname": hostName,
					key:                      diskLabel,
				},
				Namespace: "openebs",
			},
			Spec: openebscore.DeviceSpec{
				Details: openebscore.DeviceDetails{
					DeviceType: "disk",
				},
				DevLinks: []openebscore.DeviceDevLink{
					{
						Kind:  "by-id",
						Links: []string{path1},
					},
					{
						Kind:  "by-path",
						Links: []string{path2},
					},
				},
				Partitioned: "NO",
				Capacity: openebscore.DeviceCapacity{
					Storage: 10737418240,
				},
			},
			Status: openebscore.DeviceStatus{
				State: openebscore.BlockDeviceActive,
			},
		}
		_, err := f.openebsClient.OpenebsV1alpha1().BlockDevices("openebs").Create(bdObj)
		if err != nil {
			klog.Error(err)
			continue
		}
	}
}

// fakeNodeCreator creates the node CR
func (f *fixture) fakeNodeCreator(nodeName string) {
	node := &corev1.Node{}
	node.Name = nodeName
	labels := make(map[string]string)
	labels["kubernetes.io/hostname"] = node.Name
	node.Labels = labels
	_, err := f.k8sClient.CoreV1().Nodes().Create(node)
	if err != nil {
		klog.Error(err)
	}
}

// createBlockDeviceClaim creates blockdeviceclaim in similar manner how cspc
// controller creates claim
func (f *fixture) createBlockDeviceClaim(
	blockDeviceName, cspcName string, annotations map[string]string) (*openebscore.BlockDeviceClaim, error) {
	hostLabel := "kubernetes.io/hostname"
	bdObj, err := f.openebsClient.
		OpenebsV1alpha1().
		BlockDevices("openebs").
		Get(blockDeviceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	hostName := bdObj.Labels[hostLabel]
	bdcObj := &openebscore.BlockDeviceClaim{
		TypeMeta: metav1.TypeMeta{
			Kind: "BlockDeviceClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blockdeviceclaim-" + bdObj.Name,
			UID:       k8stypes.UID("bdctest" + hostName + bdObj.Name),
			Namespace: "openebs",
			Labels: map[string]string{
				string(types.CStorPoolClusterLabelKey): cspcName,
			},
			Annotations: annotations,
		},
		Spec: openebscore.DeviceClaimSpec{
			BlockDeviceName: blockDeviceName,
			BlockDeviceNodeAttributes: openebscore.BlockDeviceNodeAttributes{
				HostName: hostName,
			},
		},
	}
	return f.openebsClient.OpenebsV1alpha1().BlockDeviceClaims("openebs").Create(bdcObj)
}

// claimBlcokDevice will bound blockdevice with corresponding blockdeviceclaim
func (f *fixture) claimBlockdevice(
	bdName string, bdc *openebscore.BlockDeviceClaim) error {
	bd, err := f.openebsClient.
		OpenebsV1alpha1().
		BlockDevices("openebs").
		Get(bdName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	bd.Status.ClaimState = openebscore.BlockDeviceClaimed
	bd.Spec.ClaimRef = &corev1.ObjectReference{
		Kind:      "BlockDeviceClaim",
		Name:      bdc.Name,
		Namespace: "openebs",
	}
	_, err = f.openebsClient.OpenebsV1alpha1().BlockDevices("openebs").Update(bd)
	return err
}

// createClaimsForRaidGroupBlockDevices creates claim for blockdevices present in raid groups
func (f *fixture) createClaimsForRaidGroupBlockDevices(raidGroups []cstor.RaidGroup, cspcName string) error {
	for _, rg := range raidGroups {
		for _, bd := range rg.CStorPoolInstanceBlockDevices {
			bdcObj, err := f.createBlockDeviceClaim(bd.BlockDeviceName, cspcName, nil)
			if err != nil {
				return err
			}
			err = f.claimBlockdevice(bd.BlockDeviceName, bdcObj)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// prepareCSPIForDeploying itterate over all the blockdevices present
// in CSPI and create claims for all the blockdevice
func (f *fixture) prepareCSPIForDeploying(cspi *cstor.CStorPoolInstance) error {
	cspcName := cspi.GetLabels()[string(types.CStorPoolClusterLabelKey)]
	err := f.createClaimsForRaidGroupBlockDevices(cspi.Spec.DataRaidGroups, cspcName)
	if err != nil {
		return errors.Wrapf(err, "failed to claim for data RaidGroup blockdevice")
	}
	err = f.createClaimsForRaidGroupBlockDevices(cspi.Spec.WriteCacheRaidGroups, cspcName)
	if err != nil {
		return errors.Wrapf(err, "failed to claim blockdevice")
	}
	return nil
}

// replaceBlockDevices replace old blockdevice with new blockdevice
func replaceBlockDevices(cspi *cstor.CStorPoolInstance, oldToNewBlockDeviceMap map[string]string) {
	// Replace old blockdevice with new blockdevice if exist in Data RaidGroup
	for rgIndex, _ := range cspi.Spec.DataRaidGroups {
		for bdIndex, cspiBD := range cspi.Spec.DataRaidGroups[rgIndex].CStorPoolInstanceBlockDevices {
			if newBDName, ok := oldToNewBlockDeviceMap[cspiBD.BlockDeviceName]; ok {
				cspi.Spec.DataRaidGroups[rgIndex].
					CStorPoolInstanceBlockDevices[bdIndex].
					BlockDeviceName = newBDName
			}
		}
	}
	// Replace old blockdevice with new blockdevice if exist in WriteCache RaidGroup
	for rgIndex, _ := range cspi.Spec.WriteCacheRaidGroups {
		for bdIndex, cspiBD := range cspi.Spec.WriteCacheRaidGroups[rgIndex].CStorPoolInstanceBlockDevices {
			if newBDName, ok := oldToNewBlockDeviceMap[cspiBD.BlockDeviceName]; ok {
				cspi.Spec.WriteCacheRaidGroups[rgIndex].
					CStorPoolInstanceBlockDevices[bdIndex].
					BlockDeviceName = newBDName
			}
		}
	}
}

// updateCSPIToPerformDay2Operation updates the CSPI with provided
// configuration to perform Day2Operations
func (f *fixture) updateCSPIToPerformDay2Operation(cspiName string, tConfig testConfig) error {
	ns, name, err := cache.SplitMetaNamespaceKey(cspiName)
	if err != nil {
		return err
	}
	cspiObj, err := f.openebsClient.CstorV1().CStorPoolInstances(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	cspcName := cspiObj.GetLabels()[string(types.CStorPoolClusterLabelKey)]

	if tConfig.dataRaidGroups != nil {
		if cspiObj.Spec.PoolConfig.DataRaidGroupType == string(cstor.PoolStriped) {
			// If pool type is stripe pool then it is expected thatnew disks should be in first raid group only
			cspiObj.Spec.DataRaidGroups[0].CStorPoolInstanceBlockDevices = append(
				cspiObj.Spec.DataRaidGroups[0].CStorPoolInstanceBlockDevices,
				tConfig.dataRaidGroups[0].CStorPoolInstanceBlockDevices...)
		} else {
			// If pool type is other than mirror then append it to the raid groups
			cspiObj.Spec.DataRaidGroups = append(cspiObj.Spec.DataRaidGroups, tConfig.dataRaidGroups...)
		}
		// Claim Newely Added DataRaidGroups BlockDevices
		err = f.createClaimsForRaidGroupBlockDevices(tConfig.dataRaidGroups, cspcName)
		if err != nil {
			return err
		}
	}

	if tConfig.writeCacheRaidGroups != nil {
		// User requested for writeCache raidgroup creation after pool creation
		if cspiObj.Spec.PoolConfig.WriteCacheGroupType == "" {
			cspiObj.Spec.PoolConfig.WriteCacheGroupType = tConfig.writeCacheGroupType
		}
		if cspiObj.Spec.PoolConfig.WriteCacheGroupType == string(cstor.PoolStriped) {
			cspiObj.Spec.WriteCacheRaidGroups[0].CStorPoolInstanceBlockDevices = append(
				cspiObj.Spec.WriteCacheRaidGroups[0].CStorPoolInstanceBlockDevices,
				tConfig.writeCacheRaidGroups[0].CStorPoolInstanceBlockDevices...)
		} else {
			cspiObj.Spec.WriteCacheRaidGroups = append(cspiObj.Spec.WriteCacheRaidGroups, tConfig.writeCacheRaidGroups...)
		}
		// Claim Newely Added WriteCacheRaidGroups BlockDevices
		err = f.createClaimsForRaidGroupBlockDevices(tConfig.dataRaidGroups, cspcName)
		if err != nil {
			return err
		}
	}
	if tConfig.replaceBlcokDevices != nil {
		replaceBlockDevices(cspiObj, tConfig.replaceBlcokDevices)
	}
	_, err = f.openebsClient.CstorV1().CStorPoolInstances(ns).Update(cspiObj)
	if err != nil {
		return err
	}
	return nil
}

func (f *fixture) run(cspiName string) {
	testConfig := testConfig{
		loopCount: 1,
		loopDelay: time.Second * 0,
		poolInfo:  nil,
	}
	f.run_(cspiName, true, false, testConfig)
}

// run_ is responsible for executing sync call
func (f *fixture) run_(
	cspiName string,
	startInformers bool,
	expectError bool,
	testConfig testConfig) {
	isCSPIUpdated := false
	c, informers, recorder, err := f.newCSPIController(testConfig.poolInfo)
	if err != nil {
		f.t.Fatalf("error creating cspi controller: %v", err)
	}
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		informers.Start(stopCh)
	}

	// Waitgroup for starting pool and VolumeReplica controller goroutines.
	// var wg sync.WaitGroup
	go printEvent(recorder)

	for i := 0; i < testConfig.loopCount; i++ {
		err = c.reconcile(cspiName)
		if !expectError && err != nil {
			f.t.Errorf("error syncing cspc: %v", err)
		} else if expectError && err == nil {
			f.t.Error("expected error syncing cspc, got nil")
		}

		if testConfig.isDay2OperationNeedToPerform && !isCSPIUpdated {
			// Fill the corresponding day2 operations snippet

			err := f.updateCSPIToPerformDay2Operation(cspiName, testConfig)
			if err != nil {
				// We can do retries also
				f.t.Errorf("Failed to update CSPI %s to perform day2-operations error: %v", cspiName, err.Error())
			}
			isCSPIUpdated = true
		}

		if testConfig.loopCount > 1 {
			time.Sleep(testConfig.loopDelay)
		}
	}

}

// getBlockDeviceMapFromRaidGroups returns the map of blockdevices and value as false
func getBlockDeviceMapFromRaidGroups(rgs []cstor.RaidGroup) map[string]bool {
	bdMap := map[string]bool{}
	for _, rg := range rgs {
		for _, cspiBD := range rg.CStorPoolInstanceBlockDevices {
			bdMap[cspiBD.BlockDeviceName] = false
		}
	}
	return bdMap
}

// verifyDeviceLinksInRaidGroups verifis whether CSPI is updated with correct device
// link or not
func (f *fixture) verifyDeviceLinksInRaidGroups(
	rgs []cstor.RaidGroup, newBlockDeviceMap map[string]bool) error {
	for _, rg := range rgs {
		for _, cspiBD := range rg.CStorPoolInstanceBlockDevices {
			bdObj, err := f.openebsClient.OpenebsV1alpha1().BlockDevices("openebs").Get(cspiBD.BlockDeviceName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// Since we are using only first link in pool creation we can compare with first link in BD
			if bdObj.Spec.DevLinks[0].Links[0] != cspiBD.DevLink {
				return errors.Errorf(
					"Expected devlink %s but got devlink %s in BD: %s",
					bdObj.Spec.DevLinks[0].Links[0],
					cspiBD.DevLink,
					cspiBD.BlockDeviceName,
				)
			}
			newBlockDeviceMap[cspiBD.BlockDeviceName] = true
		}
	}
	return nil
}

// isAllBlockDevicesMarked checks whether value is true for all keys
// It is mainly to check Day2operation is successfull or not
func isAllBlockDevicesMarked(bdMap map[string]bool) (bool, string) {
	for bd, val := range bdMap {
		if !val {
			return false, fmt.Sprintf("BlockDevice %s doesn't exist", bd)
		}
	}
	return true, ""
}

// verifyCSPIAutoGeneratedSpec verifies whether cspi is
// updated with correct device links
func (f *fixture) verifyCSPIAutoGeneratedSpec(
	cspi *cstor.CStorPoolInstance, tConfig testConfig) error {
	newDataBlockDeviceMap := getBlockDeviceMapFromRaidGroups(tConfig.dataRaidGroups)
	err := f.verifyDeviceLinksInRaidGroups(cspi.Spec.DataRaidGroups, newDataBlockDeviceMap)
	if err != nil {
		return err
	}

	newWriteCacheBlockDeviceMap := getBlockDeviceMapFromRaidGroups(tConfig.writeCacheRaidGroups)
	err = f.verifyDeviceLinksInRaidGroups(cspi.Spec.WriteCacheRaidGroups, newWriteCacheBlockDeviceMap)
	if err != nil {
		return err
	}

	if tConfig.isDay2OperationNeedToPerform {
		if val, msg := isAllBlockDevicesMarked(newDataBlockDeviceMap); !val {
			return errors.Errorf("Expansion on CSPI %s has failed: %s", cspi.Name, msg)
		}
		if val, msg := isAllBlockDevicesMarked(newWriteCacheBlockDeviceMap); !val {
			return errors.Errorf("Expansion on CSPI %s has failed: %s", cspi.Name, msg)
		}
		// TODO: Validations for replacement operation
	}
	return nil
}

// verifyCSPI status verifies whether status of CSPI and returns
// error if any of the fileds are zero
func (f *fixture) verifyCSPIStatus(
	cspi *cstor.CStorPoolInstance, tConfig testConfig) error {
	if cspi.Status.Phase == "" {
		return errors.Errorf("CSPI %s phase is empty", cspi.Name)
	}
	if cspi.Status.Capacity.Total.IsZero() ||
		cspi.Status.Capacity.Used.IsZero() ||
		cspi.Status.Capacity.Free.IsZero() {
		return errors.Errorf("CSPI %s capacity field is empty %v", cspi.Name, cspi.Status.Capacity)
	}
	if tConfig.isDay2OperationNeedToPerform {
		// Add code for verifying conditions on CSPI relevant to operation
	}
	return nil
}

// printEvent prints the events reported by controller
func printEvent(recorder *record.FakeRecorder) {
	for {
		msg, ok := <-recorder.Events
		// Channel is closed
		if !ok {
			break
		}
		fmt.Println("Event: ", msg)
	}
}

/* -------------------------------------------------------------------------------------------------------------
   |                                                                                                            |
   |                                                                                                            |
   |                                   **NON-PROVISIONING TEST**                                                |
   |                                                                                                            |
   -------------------------------------------------------------------------------------------------------------
*/

// TestCSPIFinalizerAdd tests the adds the pool protection finalizer
// when a brand new cspi is created
func TestCSPIFinalizerAdd(t *testing.T) {
	f := newFixture(t)
	cspi := cstor.NewCStorPoolInstance().
		WithName("cspi-foo").
		WithNamespace("openebs")
	cspi.Kind = "cstorpoolinstance"
	f.cspiLister = append(f.cspiLister, cspi)
	f.openebsObjects = append(f.openebsObjects, cspi)
	f.SetFakeClient()
	os.Setenv(string(common.OpenEBSIOPoolName), "1234")

	common.Init()
	f.run(testutil.GetKey(cspi, t))

	os.Unsetenv(string(common.OpenEBSIOPoolName))
	cspi, err := f.openebsClient.CstorV1().CStorPoolInstances(cspi.Namespace).Get(cspi.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("error getting cspc %s: %v", cspi.Name, err)
	}
	if !cspi.HasFinalizer(types.PoolProtectionFinalizer) {
		t.Errorf("expected finalizer %s on cspi %s but was not found", types.PoolProtectionFinalizer, cspi.Name)
	}
}

// TestCSPIFinalizerRemoval tests the rmoval of pool protection finalizer
func TestCSPIFinalizerRemoval(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	tests := map[string]struct {
		cspi                 *cstor.CStorPoolInstance
		shouldFinalizerExist bool
		testConfig           testConfig
		expectError          bool
	}{
		"When pool doesn't exist deletion of CSPI is triggered": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-1").
				WithNamespace("openebs").
				WithFinalizer(types.PoolProtectionFinalizer),
			testConfig: testConfig{
				loopCount: 2,
				loopDelay: time.Second * 0,
				poolInfo:  &zpool.MockPoolInfo{},
			},
			shouldFinalizerExist: false,
			expectError:          false,
		},
		"When pool exist deletion of CSPI is triggered": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-2").
				WithNamespace("openebs").
				WithFinalizer(types.PoolProtectionFinalizer),
			testConfig: testConfig{
				loopCount: 2,
				loopDelay: time.Second * 0,
				poolInfo: &zpool.MockPoolInfo{
					PoolName: "cstor-1234",
				},
			},
			shouldFinalizerExist: false,
			expectError:          false,
		},
		"When deletion of pool is failed": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-3").
				WithNamespace("openebs").
				WithFinalizer(types.PoolProtectionFinalizer),
			testConfig: testConfig{
				loopCount: 2,
				loopDelay: time.Second * 0,
				poolInfo: &zpool.MockPoolInfo{
					PoolName: "cstor-1234",
					TestConfig: zpool.TestConfig{
						ZpoolCommand: zpool.ZpoolCommandError{
							ZpoolDestroyError: true,
						},
					},
				},
			},
			shouldFinalizerExist: true,
			expectError:          true,
		},
	}
	os.Setenv(string(common.OpenEBSIOPoolName), "1234")
	common.Init()
	for name, test := range tests {
		name := name
		test := test
		test.cspi.Kind = "cstorpoolinstance"
		test.cspi.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		t.Run(name, func(t *testing.T) {
			// Create a CSPI to persist it in a fake store
			f.openebsClient.CstorV1().CStorPoolInstances("openebs").Create(test.cspi)

			f.run_(testutil.GetKey(test.cspi, t), true, test.expectError, test.testConfig)

			cspi, err := f.openebsClient.CstorV1().CStorPoolInstances(test.cspi.Namespace).Get(test.cspi.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("error getting cspc %s: %v", cspi.Name, err)
			}
			if cspi.HasFinalizer(types.PoolProtectionFinalizer) != test.shouldFinalizerExist {
				t.Errorf(
					"%q test failed %s finalizer existence on %s cspi expected: %t but got: %t",
					name,
					types.PoolProtectionFinalizer,
					cspi.Name,
					test.shouldFinalizerExist,
					cspi.HasFinalizer(types.PoolProtectionFinalizer),
				)
			}
		})
	}
	os.Unsetenv(string(common.OpenEBSIOPoolName))
}

/* -------------------------------------------------------------------------------------------------------------
   |                                                                                                            |
   |                                                                                                            |
   |                                   **PROVISIONING TEST**                                                    |
   |                                                                                                            |
   |                                                                                                            |
   -------------------------------------------------------------------------------------------------------------
*/

// TestCSPIPoolProvisioning tests the pool provisioning
// with various combinations of RaidGroups
func TestCSPIPoolProvisioning(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	f.createFakeBlockDevices(50, "node1")
	f.fakeNodeCreator("node1")

	tests := map[string]struct {
		cspi                         *cstor.CStorPoolInstance
		shouldVerifyCSPIAutoGenerate bool
		shouldVerifyCSPIStatus       bool
		testConfig                   testConfig
	}{
		"Stripe Pool Provisioning Without WriteCache RaidGroup": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-stripe-data").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo-stripe"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("stripe")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-1"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.MockPoolInfo{},
			},
		},
		"Stripe Pool Provisioning With WriteCache RaidGroup": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-stripe-writecache").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo-stripe-writecache"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("stripe").
					WithWriteCacheGroupType("stripe"),
				).
				WithDataRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-2"},
						},
					},
				}).
				WithWriteCacheRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-3"},
					}},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.MockPoolInfo{},
			},
		},
		"Stripe Pool Provisioning With Multiple DataRaidGroups & WriteCache RaidGroup": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-stripe-multiple-raidgroup").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo-stripe-multiple-raidgroups"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("stripe").
					WithWriteCacheGroupType("stripe"),
				).
				WithDataRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-4"},
							{BlockDeviceName: "blockdevice-5"},
							{BlockDeviceName: "blockdevice-6"},
						},
					},
				}).
				WithWriteCacheRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-7"},
						{BlockDeviceName: "blockdevice-8"},
						{BlockDeviceName: "blockdevice-9"},
					}},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.MockPoolInfo{},
			},
		},
		"Mirror Pool With Multiple RaidGroups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-mirror").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo-mirror"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("mirror").
					WithWriteCacheGroupType("mirror"),
				).
				WithDataRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-10"},
							{BlockDeviceName: "blockdevice-11"},
						},
					},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-12"},
							{BlockDeviceName: "blockdevice-13"},
						},
					},
				}).
				WithWriteCacheRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-14"},
							{BlockDeviceName: "blockdevice-15"},
						},
					},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-16"},
							{BlockDeviceName: "blockdevice-17"},
						},
					},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.MockPoolInfo{},
			},
		},
		"Raidz Pool With Multiple Data RaidGroups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-raidz").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo-raidz"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("raidz").
					WithWriteCacheGroupType("stripe"),
				).
				WithDataRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-18"},
							{BlockDeviceName: "blockdevice-19"},
							{BlockDeviceName: "blockdevice-20"},
						},
					},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-21"},
							{BlockDeviceName: "blockdevice-22"},
							{BlockDeviceName: "blockdevice-23"},
						},
					},
				}).
				WithWriteCacheRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-24"},
							{BlockDeviceName: "blockdevice-25"},
						},
					},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.MockPoolInfo{},
			},
		},
		"Raidz2 Pool With Multiple Data RaidGroups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-raidz2").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo-raidz2"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("raidz2").
					WithWriteCacheGroupType("mirror"),
				).
				WithDataRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-26"},
							{BlockDeviceName: "blockdevice-27"},
							{BlockDeviceName: "blockdevice-28"},
							{BlockDeviceName: "blockdevice-29"},
							{BlockDeviceName: "blockdevice-30"},
							{BlockDeviceName: "blockdevice-31"},
						},
					},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-32"},
							{BlockDeviceName: "blockdevice-33"},
							{BlockDeviceName: "blockdevice-34"},
							{BlockDeviceName: "blockdevice-35"},
							{BlockDeviceName: "blockdevice-36"},
							{BlockDeviceName: "blockdevice-37"},
						},
					},
				}).
				WithWriteCacheRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-38"},
							{BlockDeviceName: "blockdevice-39"},
						},
					},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.MockPoolInfo{},
			},
		},
	}

	os.Setenv(string(common.OpenEBSIOPoolName), "1234")
	os.Setenv(util.Namespace, "openebs")
	common.Init()
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			test.cspi.Kind = "CStorPoolInstance"
			// Create a CSPI to persist it in a fake store
			f.openebsClient.CstorV1().CStorPoolInstances("openebs").Create(test.cspi)
			// Create claims for blockdevices exist on cspi
			err := f.prepareCSPIForDeploying(test.cspi)
			if err != nil {
				t.Errorf("Test: %q failed to prepare pools %s", name, err.Error())
			}
			f.run_(testutil.GetKey(test.cspi, t), true, false, test.testConfig)
			// CSPI controller is to create pools and manage it using zpool/zfs command
			// line utility. Since there is no real zrepl process is running we can
			// check spec and status autogenerated part
			cspi, err := f.openebsClient.
				CstorV1().
				CStorPoolInstances(test.cspi.Namespace).
				Get(test.cspi.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("error getting cspc %s: %v", cspi.Name, err)
			}
			// Verify does AutoGenerated Configuration was updated(As of now
			// we are verifying only device links)
			if test.shouldVerifyCSPIAutoGenerate {
				err = f.verifyCSPIAutoGeneratedSpec(cspi, test.testConfig)
				if err != nil {
					t.Errorf("Test: %q validation failed %s", name, err.Error())
				}
			}
			// Verify CSPI Status Part
			if test.shouldVerifyCSPIStatus {
				err = f.verifyCSPIStatus(cspi, test.testConfig)
				if err != nil {
					t.Errorf("Test: %q validation failed %s", name, err.Error())
				}
			}
		})
	}
	os.Unsetenv(string(common.OpenEBSIOPoolName))
	os.Unsetenv(util.Namespace)
}

/* -------------------------------------------------------------------------------------------------------------
   |                                                                                                            |
   |                                                                                                            |
   |                                   **Day2 Operations**                                                      |
   |                                                                                                            |
   |                                                                                                            |
   -------------------------------------------------------------------------------------------------------------
*/

// TestCSPIPoolExpansion will provision pool and then it
// will perform pool expansion by adding blockdevices
func TestCSPIPoolExpansion(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	f.createFakeBlockDevices(50, "node1")
	f.fakeNodeCreator("node1")

	tests := map[string]struct {
		cspi                         *cstor.CStorPoolInstance
		shouldVerifyCSPIAutoGenerate bool
		shouldVerifyCSPIStatus       bool
		testConfig                   testConfig
	}{
		"Provision Stripe Pool And Expand DataRaidGroup": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-stripe-expand").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo-stripe-expand"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("stripe")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-1"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: testConfig{
				loopCount:                    4,
				loopDelay:                    time.Microsecond * 100,
				poolInfo:                     &zpool.MockPoolInfo{},
				isDay2OperationNeedToPerform: true,
				dataRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-2"},
						{BlockDeviceName: "blockdevice-3"}},
					},
				},
			},
		},
	}

	os.Setenv(string(common.OpenEBSIOPoolName), "1234")
	os.Setenv(util.Namespace, "openebs")
	common.Init()
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			test.cspi.Kind = "CStorPoolInstance"
			// Create a CSPI to persist it in a fake store
			f.openebsClient.CstorV1().CStorPoolInstances("openebs").Create(test.cspi)
			// Create claims for blockdevices exist on cspi
			err := f.prepareCSPIForDeploying(test.cspi)
			if err != nil {
				t.Errorf("Test: %q failed to prepare pools %s", name, err.Error())
			}
			f.run_(testutil.GetKey(test.cspi, t), true, false, test.testConfig)
			// CSPI controller is to create pools and manage it using zpool/zfs command
			// line utility. Since there is no real zrepl process is running we can
			// check spec and status autogenerated part
			cspi, err := f.openebsClient.
				CstorV1().
				CStorPoolInstances(test.cspi.Namespace).
				Get(test.cspi.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("error getting cspc %s: %v", cspi.Name, err)
			}
			// Verify does AutoGenerated Configuration was updated(As of now
			// we are verifying only device links)
			if test.shouldVerifyCSPIAutoGenerate {
				err = f.verifyCSPIAutoGeneratedSpec(cspi, test.testConfig)
				if err != nil {
					t.Errorf("Test: %q validation failed %s", name, err.Error())
				}
			}
			// Verify CSPI Status Part
			if test.shouldVerifyCSPIStatus {
				err = f.verifyCSPIStatus(cspi, test.testConfig)
				if err != nil {
					t.Errorf("Test: %q validation failed %s", name, err.Error())
				}
			}
		})
	}
	os.Unsetenv(string(common.OpenEBSIOPoolName))
	os.Unsetenv(util.Namespace)
}
