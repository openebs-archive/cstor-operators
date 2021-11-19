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
	"fmt"
	"html"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebscore "github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	"github.com/openebs/api/v3/pkg/apis/types"
	openebsFakeClientset "github.com/openebs/api/v3/pkg/client/clientset/versioned/fake"
	openebsinformers "github.com/openebs/api/v3/pkg/client/informers/externalversions"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/openebs/cstor-operators/pkg/controllers/common"
	cspiutil "github.com/openebs/cstor-operators/pkg/controllers/cspi-controller/util"
	"github.com/openebs/cstor-operators/pkg/controllers/testutil"
	executor "github.com/openebs/cstor-operators/pkg/controllers/testutil/zcmd/executor"
	zfs "github.com/openebs/cstor-operators/pkg/controllers/testutil/zcmd/zfs"
	zpool "github.com/openebs/cstor-operators/pkg/controllers/testutil/zcmd/zpool"
	"github.com/openebs/cstor-operators/pkg/version"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
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

	// Actions expected to happen on the client. Objects from here are also
	// preloaded into NewSimpleFake.
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
	// replaceBlockDevices in cStor pool
	replaceBlockDevices map[string]string
	// isDay2OperationNeedToPerform is set then above operations will be performed
	isDay2OperationNeedToPerform bool
	// loopCount times reconcile function will be called
	loopCount int
	// ejectErrorCount eject the error in zpool commands once it reaches to zero
	// NOTE: If test needs to have error then ejectErrorCount > loopCount
	ejectErrorCount int
	// time interval to trigger reconciliation
	loopDelay time.Duration
	// poolInfo will usefull to execute zpool commands
	poolInfo *zpool.PoolMocker
	// volumeInfo will usefull to execute zfs commands
	volumeInfo *zfs.VolumeMocker
	// shouldPoolOperationInProgress it can be set to true only when
	// errors were injected in ZPOOL commands. If the value is enabled
	// it will verify whether the pool operations are in porgress or not
	// if the pool operations are not in progress then test will be marked
	// as failed
	shouldPoolOperationInProgress bool
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
	testConfig *testConfig) (*CStorPoolInstanceController, openebsinformers.SharedInformerFactory, error) {
	//// Load kubernetes client set by preloading with k8s objects.
	//f.k8sClient = fake.NewSimpleClientset(f.k8sObjects...)

	//// Load openebs client set by preloading with openebs objects.
	//f.openebsClient = openebsFakeClientset.NewSimpleClientset(f.openebsObjects...)
	if testConfig.volumeInfo == nil {
		testConfig.volumeInfo = &zfs.VolumeMocker{}
	}

	fakeZCMDExecutor := executor.NewFakeZCommandFromMockers(testConfig.poolInfo, testConfig.volumeInfo)

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
	return controller, cspiInformerFactory, nil
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
		_, err := f.openebsClient.OpenebsV1alpha1().BlockDevices("openebs").Create(context.TODO(), bdObj, metav1.CreateOptions{})
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
	_, err := f.k8sClient.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
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
		Get(context.TODO(), blockDeviceName, metav1.GetOptions{})
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
	return f.openebsClient.OpenebsV1alpha1().BlockDeviceClaims("openebs").Create(context.TODO(), bdcObj, metav1.CreateOptions{})
}

// claimBlcokDevice will bound blockdevice with corresponding blockdeviceclaim
func (f *fixture) claimBlockdevice(
	bdName string, bdc *openebscore.BlockDeviceClaim) error {
	bd, err := f.openebsClient.
		OpenebsV1alpha1().
		BlockDevices("openebs").
		Get(context.TODO(), bdName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	bd.Status.ClaimState = openebscore.BlockDeviceClaimed
	bd.Spec.ClaimRef = &corev1.ObjectReference{
		Kind:      "BlockDeviceClaim",
		Name:      bdc.Name,
		Namespace: "openebs",
	}
	_, err = f.openebsClient.OpenebsV1alpha1().BlockDevices("openebs").Update(context.TODO(), bd, metav1.UpdateOptions{})
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
func (f *fixture) replaceBlockDevices(
	cspi *cstor.CStorPoolInstance,
	oldToNewBlockDeviceMap map[string]string) error {
	cspcName := cspi.GetLabels()[string(types.CStorPoolClusterLabelKey)]
	// Replace old blockdevice with new blockdevice if exist in Data RaidGroup
	for rgIndex := range cspi.Spec.DataRaidGroups {
		for bdIndex, cspiBD := range cspi.Spec.DataRaidGroups[rgIndex].CStorPoolInstanceBlockDevices {
			if newBDName, ok := oldToNewBlockDeviceMap[cspiBD.BlockDeviceName]; ok {
				cspi.Spec.DataRaidGroups[rgIndex].
					CStorPoolInstanceBlockDevices[bdIndex].
					BlockDeviceName = newBDName
			}
		}
	}
	// Replace old blockdevice with new blockdevice if exist in WriteCache RaidGroup
	for rgIndex := range cspi.Spec.WriteCacheRaidGroups {
		for bdIndex, cspiBD := range cspi.Spec.WriteCacheRaidGroups[rgIndex].CStorPoolInstanceBlockDevices {
			if newBDName, ok := oldToNewBlockDeviceMap[cspiBD.BlockDeviceName]; ok {
				cspi.Spec.WriteCacheRaidGroups[rgIndex].
					CStorPoolInstanceBlockDevices[bdIndex].
					BlockDeviceName = newBDName
			}
		}
	}

	// Claim New BlockDevices witch replacement marks
	for oldBDName, newBDName := range oldToNewBlockDeviceMap {
		bdcObj, err := f.createBlockDeviceClaim(newBDName, cspcName, map[string]string{types.PredecessorBDLabelKey: oldBDName})
		if err != nil {
			return err
		}
		err = f.claimBlockdevice(newBDName, bdcObj)
		if err != nil {
			return err
		}
	}
	return nil
}

// updateCSPIToPerformDay2Operation updates the CSPI with provided
// configuration to perform Day2Operations
func (f *fixture) updateCSPIToPerformDay2Operation(cspiName string, tConfig testConfig) error {
	ns, name, err := cache.SplitMetaNamespaceKey(cspiName)
	if err != nil {
		return err
	}
	cspiObj, err := f.openebsClient.CstorV1().CStorPoolInstances(ns).Get(context.TODO(), name, metav1.GetOptions{})
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
			// If writeCacheRaidGroup is nil initilize it
			if cspiObj.Spec.WriteCacheRaidGroups == nil {
				cspiObj.Spec.WriteCacheRaidGroups = []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{}}}
			}
			cspiObj.Spec.WriteCacheRaidGroups[0].CStorPoolInstanceBlockDevices = append(
				cspiObj.Spec.WriteCacheRaidGroups[0].CStorPoolInstanceBlockDevices,
				tConfig.writeCacheRaidGroups[0].CStorPoolInstanceBlockDevices...)
		} else {
			cspiObj.Spec.WriteCacheRaidGroups = append(cspiObj.Spec.WriteCacheRaidGroups, tConfig.writeCacheRaidGroups...)
		}
		// Claim Newely Added WriteCacheRaidGroups BlockDevices
		err = f.createClaimsForRaidGroupBlockDevices(tConfig.writeCacheRaidGroups, cspcName)
		if err != nil {
			return err
		}
	}
	if tConfig.replaceBlockDevices != nil {
		err = f.replaceBlockDevices(cspiObj, tConfig.replaceBlockDevices)
		if err != nil {
			return err
		}
	}
	_, err = f.openebsClient.CstorV1().CStorPoolInstances(ns).Update(context.TODO(), cspiObj, metav1.UpdateOptions{})
	return err
}

func (f *fixture) run(cspiName string) {
	testConfig := testConfig{
		loopCount: 1,
		loopDelay: time.Second * 0,
		poolInfo:  nil,
	}
	f.run_(cspiName, true, false, &testConfig)
}

// run_ is responsible for executing sync call
func (f *fixture) run_(
	cspiName string,
	startInformers bool,
	expectError bool,
	testConfig *testConfig) {
	isCSPIUpdated := false
	ejectErrorCount := testConfig.ejectErrorCount
	c, informers, err := f.newCSPIController(testConfig)
	if err != nil {
		f.t.Fatalf("error creating cspi controller: %v", err)
	}
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		informers.Start(stopCh)
	}
	defer func(recorderInterface record.EventRecorder) {
		recorder := recorderInterface.(*record.FakeRecorder)
		close(recorder.Events)
	}(c.recorder)

	// Waitgroup for starting pool and VolumeReplica controller goroutines.
	// var wg sync.WaitGroup
	go printEvent(c.recorder)

	for i := 0; i < testConfig.loopCount; i++ {

		// TODO: Need to check with team how feasible injecting errors and ejecting them
		if testConfig.poolInfo != nil && ejectErrorCount <= 0 {
			// Eject all zpool command errors which were inserted during test configuration
			// time
			testConfig.poolInfo.TestConfig.ZpoolCommand = zpool.ZpoolCommandError{}
			// For pool testcases volumeInfo mightnot be initilized
			if testConfig.volumeInfo != nil {
				testConfig.volumeInfo.TestConfig.ZFSCommand = zfs.ZFSCommandError{}
			}
		}

		err = c.reconcile(cspiName)
		if !expectError && err != nil {
			f.t.Errorf("error syncing cspc: %v", err)
		} else if expectError && err == nil {
			f.t.Error("expected error syncing cspc, got nil")
		}

		// When CSPI is updated to perform pool operation and if errors are
		// injected then we can verify pool operation progress
		if isCSPIUpdated && testConfig.shouldPoolOperationInProgress && ejectErrorCount > 0 {
			// When error is injected pool operations should be in pending state
			if ok, msg := f.isPoolOperationPending(cspiName, testConfig); !ok {
				f.t.Errorf("injected error in zpool commands %s", msg)
			}
		}

		if testConfig.isDay2OperationNeedToPerform && !isCSPIUpdated {
			// Fill the corresponding day2 operations snippet

			err := f.updateCSPIToPerformDay2Operation(cspiName, *testConfig)
			if err != nil {
				// We can do retries also
				f.t.Errorf("Failed to update CSPI %s to perform day2-operations error: %v", cspiName, err.Error())
			}
			isCSPIUpdated = true
		}

		ejectErrorCount--
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
			bdObj, err := f.openebsClient.OpenebsV1alpha1().BlockDevices("openebs").Get(context.TODO(), cspiBD.BlockDeviceName, metav1.GetOptions{})
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
	cspi *cstor.CStorPoolInstance, tConfig *testConfig) error {
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

func isStatusConditionMatched(
	cspi *cstor.CStorPoolInstance,
	condType cstor.CStorPoolInstanceConditionType,
	reason string) (bool, string) {
	cond := cspiutil.GetCSPICondition(cspi.Status, condType)
	if cond == nil {
		return false, fmt.Sprintf("%s condition not exist", condType)
	}
	if cond.Reason != reason {
		return false, fmt.Sprintf(
			"%s reason is expected to present on %s status condition but got %s reason",
			reason,
			condType,
			cond.Reason,
		)
	}
	return true, ""
}

// verifyCSPI status verifies whether status of CSPI and returns
// error if any of the fileds are zero
func (f *fixture) verifyCSPIStatus(
	cspi *cstor.CStorPoolInstance, tConfig *testConfig) error {
	if cspi.Status.Phase == "" {
		return errors.Errorf("CSPI %s phase is empty", cspi.Name)
	}
	if cspi.Status.Capacity.Total.IsZero() ||
		cspi.Status.Capacity.Used.IsZero() ||
		cspi.Status.Capacity.Free.IsZero() {
		return errors.Errorf("CSPI %s capacity field is empty %v", cspi.Name, cspi.Status.Capacity)
	}
	if tConfig.isDay2OperationNeedToPerform {
		if tConfig.dataRaidGroups != nil || tConfig.writeCacheRaidGroups != nil {
			if ok, msg := isStatusConditionMatched(cspi, cstor.CSPIPoolExpansion, "PoolExpansionSuccessful"); !ok {
				return errors.Errorf("CSPI %s pool expansion condition %s", cspi.Name, msg)
			}
		}
		if tConfig.replaceBlockDevices != nil {
			if ok, msg := isStatusConditionMatched(cspi, cstor.CSPIDiskReplacement, "BlockDeviceReplacementSucceess"); !ok {
				return errors.Errorf("CSPI %s pool replacement condtion %s", cspi.Name, msg)
			}
		}
	}
	return nil
}

// isPoolOperationPending returns true if pool operation is InProgress else return false
func (f *fixture) isPoolOperationPending(cspiName string, tConfig *testConfig) (bool, string) {
	ns, name, err := cache.SplitMetaNamespaceKey(cspiName)
	if err != nil {
		return false, err.Error()
	}
	cspiObj, err := f.openebsClient.CstorV1().CStorPoolInstances(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false, err.Error()
	}
	if tConfig.writeCacheRaidGroups != nil || tConfig.dataRaidGroups != nil {
		if ok, msg := isStatusConditionMatched(cspiObj, cstor.CSPIPoolExpansion, "PoolExpansionInProgress"); !ok {
			return false, fmt.Sprintf("Expected pool expansion to be in progress but %s", msg)
		}
	}
	return true, ""
}

// isReplacementMarksExists returns false when there are no replacement marks
// 1. Verify existence of claim for Old BlockDevice
// 2. Verify existence of "openebs.io/bd-predecessor" annotation on new blockdevice claim
func (f *fixture) isReplacementMarksExists(testConfig testConfig) (bool, string) {
	// Old BDC should be deleted and new BDC shouldn't have any replacement marks
	for oldBDName, newBDName := range testConfig.replaceBlockDevices {
		// Since we are creating blockdevice claims with "blockdeviceclaim-" + blockdevice name
		oldBDCName := "blockdeviceclaim-" + oldBDName
		newBDName := "blockdeviceclaim-" + newBDName
		_, err := f.openebsClient.
			OpenebsV1alpha1().
			BlockDeviceClaims("openebs").
			Get(context.TODO(), oldBDCName, metav1.GetOptions{})
		if err != nil {
			if !k8serror.IsNotFound(err) {
				return true, fmt.Sprintf("Failed to get claim of old blockdevice %s error: %s", oldBDName, err.Error())
			}
		}
		newBDBDC, err := f.openebsClient.
			OpenebsV1alpha1().
			BlockDeviceClaims("openebs").
			Get(context.TODO(), newBDName, metav1.GetOptions{})
		if err != nil {
			return true, fmt.Sprintf("Failed to get claim of blockdevice %s error: %s", newBDName, err.Error())
		}
		if _, ok := newBDBDC.GetAnnotations()[types.PredecessorBDLabelKey]; ok {
			return true, fmt.Sprintf("Replacement mark exist on claim of blockdevice %s", newBDName)
		}
	}
	return false, ""
}

// printEvent prints the events reported by controller
func printEvent(recorderInterface record.EventRecorder) {
	recorder := recorderInterface.(*record.FakeRecorder)
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
	cspi, err := f.openebsClient.CstorV1().CStorPoolInstances(cspi.Namespace).Get(context.TODO(), cspi.Name, metav1.GetOptions{})
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
				poolInfo:  &zpool.PoolMocker{},
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
				poolInfo: &zpool.PoolMocker{
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
				poolInfo: &zpool.PoolMocker{
					PoolName: "cstor-1234",
					TestConfig: zpool.TestConfig{
						ZpoolCommand: zpool.ZpoolCommandError{
							ZpoolDestroyError: true,
						},
					},
				},
				ejectErrorCount: 3,
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
			f.openebsClient.CstorV1().CStorPoolInstances("openebs").Create(context.TODO(), test.cspi, metav1.CreateOptions{})

			f.run_(testutil.GetKey(test.cspi, t), true, test.expectError, &test.testConfig)

			cspi, err := f.openebsClient.CstorV1().CStorPoolInstances(test.cspi.Namespace).Get(context.TODO(), test.cspi.Name, metav1.GetOptions{})
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
	f.createFakeBlockDevices(80, "node1")
	f.fakeNodeCreator("node1")

	tests := map[string]struct {
		cspi                         *cstor.CStorPoolInstance
		shouldVerifyCSPIAutoGenerate bool
		shouldVerifyCSPIStatus       bool
		testConfig                   *testConfig
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
			testConfig: &testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
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
			testConfig: &testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
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
			testConfig: &testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
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
			testConfig: &testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
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
			testConfig: &testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
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
			testConfig: &testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
			},
		},
		"Raidz Pool With multiple disks in multiple Data RaidGroups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-multiple-raidz1").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-multiple-raidz1"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("raidz").
					WithWriteCacheGroupType("raidz"),
				).
				WithDataRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-40"},
							{BlockDeviceName: "blockdevice-41"},
							{BlockDeviceName: "blockdevice-42"},
							{BlockDeviceName: "blockdevice-43"},
							{BlockDeviceName: "blockdevice-44"},
						},
					},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-45"},
							{BlockDeviceName: "blockdevice-46"},
							{BlockDeviceName: "blockdevice-47"},
							{BlockDeviceName: "blockdevice-48"},
							{BlockDeviceName: "blockdevice-49"},
						},
					},
				}).
				WithWriteCacheRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-50"},
							{BlockDeviceName: "blockdevice-51"},
							{BlockDeviceName: "blockdevice-52"},
							{BlockDeviceName: "blockdevice-53"},
							{BlockDeviceName: "blockdevice-54"},
						},
					},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-55"},
							{BlockDeviceName: "blockdevice-56"},
							{BlockDeviceName: "blockdevice-57"},
							{BlockDeviceName: "blockdevice-58"},
							{BlockDeviceName: "blockdevice-59"},
						},
					},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
			},
		},
		"Raidz2 Pool With Multiple disks with multiple Data RaidGroups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-multiple-raidz2").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-multiple-raidz2"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("raidz2").
					WithWriteCacheGroupType("raidz"),
				).
				WithDataRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-60"},
							{BlockDeviceName: "blockdevice-61"},
							{BlockDeviceName: "blockdevice-62"},
							{BlockDeviceName: "blockdevice-63"},
							{BlockDeviceName: "blockdevice-64"},
							{BlockDeviceName: "blockdevice-65"},
						},
					},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-66"},
							{BlockDeviceName: "blockdevice-67"},
							{BlockDeviceName: "blockdevice-68"},
							{BlockDeviceName: "blockdevice-69"},
							{BlockDeviceName: "blockdevice-70"},
							{BlockDeviceName: "blockdevice-71"},
							{BlockDeviceName: "blockdevice-72"},
							{BlockDeviceName: "blockdevice-73"},
							{BlockDeviceName: "blockdevice-74"},
							{BlockDeviceName: "blockdevice-75"},
						},
					},
				}).
				WithWriteCacheRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-76"},
							{BlockDeviceName: "blockdevice-77"},
							{BlockDeviceName: "blockdevice-78"},
							{BlockDeviceName: "blockdevice-79"},
							{BlockDeviceName: "blockdevice-80"},
						},
					},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount: 3,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
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
			f.openebsClient.CstorV1().CStorPoolInstances("openebs").Create(context.TODO(), test.cspi, metav1.CreateOptions{})
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
				Get(context.TODO(), test.cspi.Name, metav1.GetOptions{})
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
	f.createFakeBlockDevices(60, "node1")
	f.fakeNodeCreator("node1")

	tests := map[string]struct {
		cspi                         *cstor.CStorPoolInstance
		shouldVerifyCSPIAutoGenerate bool
		shouldVerifyCSPIStatus       bool
		expectedPoolOperationPending bool
		testConfig                   *testConfig
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
			testConfig: &testConfig{
				loopCount:                    4,
				loopDelay:                    time.Microsecond * 100,
				poolInfo:                     &zpool.PoolMocker{},
				isDay2OperationNeedToPerform: true,
				dataRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-2"},
						{BlockDeviceName: "blockdevice-3"}},
					},
				},
			},
		},
		"Provision Stripe Pool and expand data and writecache raid groups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-bar-stripe-expand").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-bar-stripe-expand"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("stripe").
					WithWriteCacheGroupType("stripe")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-4"},
						{BlockDeviceName: "blockdevice-5"},
					},
				},
				}).
				WithWriteCacheRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-6"},
						{BlockDeviceName: "blockdevice-7"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount:                    4,
				loopDelay:                    time.Microsecond * 100,
				poolInfo:                     &zpool.PoolMocker{},
				isDay2OperationNeedToPerform: true,
				dataRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-8"},
						{BlockDeviceName: "blockdevice-9"}},
					},
				},
				writeCacheRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-10"},
						{BlockDeviceName: "blockdevice-11"}},
					},
				},
			},
		},
		"Provision Mirror Pool and expand writecache raid groups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-bar-mirror-expand").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-bar-mirror-expand"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("mirror")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-12"},
						{BlockDeviceName: "blockdevice-13"},
					},
				},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-14"},
							{BlockDeviceName: "blockdevice-15"},
						},
					},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount:                    4,
				loopDelay:                    time.Microsecond * 100,
				poolInfo:                     &zpool.PoolMocker{},
				isDay2OperationNeedToPerform: true,
				writeCacheGroupType:          "mirror",
				writeCacheRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-16"},
						{BlockDeviceName: "blockdevice-17"}},
					},
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-18"},
						{BlockDeviceName: "blockdevice-19"}},
					},
				},
			},
		},
		"Provision Mirror Pool and expand data raid groups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-expand-data-raidgroup").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-expand-data-raidgroup"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("mirror")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-20"},
						{BlockDeviceName: "blockdevice-21"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount:                    4,
				loopDelay:                    time.Microsecond * 100,
				poolInfo:                     &zpool.PoolMocker{},
				isDay2OperationNeedToPerform: true,
				dataRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-22"},
						{BlockDeviceName: "blockdevice-23"}},
					},
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-24"},
						{BlockDeviceName: "blockdevice-25"}},
					},
				},
			},
		},
		"Provision Mirror Pool and expand data and write cache raid groups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-expand-both-mirror-raidgroups").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-expand-both-mirror-raidgroups"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("mirror")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-26"},
						{BlockDeviceName: "blockdevice-27"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount:                    4,
				loopDelay:                    time.Microsecond * 100,
				poolInfo:                     &zpool.PoolMocker{},
				isDay2OperationNeedToPerform: true,
				dataRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-28"},
						{BlockDeviceName: "blockdevice-29"}},
					},
				},
				writeCacheGroupType: "mirror",
				writeCacheRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-30"},
						{BlockDeviceName: "blockdevice-31"}},
					},
				},
			},
		},
		"Provision Raidz Pool and expand data and write cache raid groups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-expand-both-raidz-raidgroups").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-expand-both-raidz-raidgroups"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("raidz")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-32"},
						{BlockDeviceName: "blockdevice-33"},
						{BlockDeviceName: "blockdevice-34"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount:                    4,
				loopDelay:                    time.Microsecond * 100,
				poolInfo:                     &zpool.PoolMocker{},
				isDay2OperationNeedToPerform: true,
				dataRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-35"},
						{BlockDeviceName: "blockdevice-36"},
						{BlockDeviceName: "blockdevice-37"}},
					},
				},
				writeCacheGroupType: "raidz",
				writeCacheRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-38"},
						{BlockDeviceName: "blockdevice-39"}},
					},
				},
			},
		},
		"Provision Raidz Pool and add writecache stripe raid groups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-add-writecache-raidgroup").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-add-writecache-raidgroup"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("raidz")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-40"},
						{BlockDeviceName: "blockdevice-41"},
						{BlockDeviceName: "blockdevice-42"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount:                    4,
				loopDelay:                    time.Microsecond * 100,
				poolInfo:                     &zpool.PoolMocker{},
				isDay2OperationNeedToPerform: true,
				writeCacheGroupType:          "stripe",
				writeCacheRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-43"},
						{BlockDeviceName: "blockdevice-44"}},
					},
				},
			},
		},
		"Provision Raidz2 Pool and add writecache raidz raid groups": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-add-writecache-raidz-raidgroup").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-add-writecache-raidz-raidgroup"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("raidz2")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-45"},
						{BlockDeviceName: "blockdevice-46"},
						{BlockDeviceName: "blockdevice-47"},
						{BlockDeviceName: "blockdevice-48"},
						{BlockDeviceName: "blockdevice-49"},
						{BlockDeviceName: "blockdevice-50"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount:                    4,
				loopDelay:                    time.Microsecond * 100,
				poolInfo:                     &zpool.PoolMocker{},
				isDay2OperationNeedToPerform: true,
				writeCacheGroupType:          "raidz",
				writeCacheRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-51"},
						{BlockDeviceName: "blockdevice-52"},
						{BlockDeviceName: "blockdevice-53"}},
					},
				},
			},
		},
		"Provision stripe Pool and expand data raidgroup by injecting error in zpool add command": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-stripe-expand-witherror").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo-stripe-expand-witherror"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("stripe")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-54"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount: 5,
				// In 4th iteration expanding will be successfull
				// In 5th iteration device links will be updated
				ejectErrorCount:               3,
				shouldPoolOperationInProgress: true,
				loopDelay:                     time.Microsecond * 100,
				poolInfo: &zpool.PoolMocker{
					TestConfig: zpool.TestConfig{
						ZpoolCommand: zpool.ZpoolCommandError{
							ZpoolAddError: true,
						},
					},
				},
				isDay2OperationNeedToPerform: true,
				dataRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-55"},
						{BlockDeviceName: "blockdevice-56"}},
					},
				},
			},
		},
		"Provision stripe Pool and expand data raidgroup by injecting error permanently in zpool add command": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-bar-stripe-expand-witherror").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-bar-stripe-expand-witherror"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("stripe")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-57"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: false,
			shouldVerifyCSPIStatus:       false,
			expectedPoolOperationPending: true,
			testConfig: &testConfig{
				loopCount: 4,
				// Expansion operation shouldn't be succeeded
				ejectErrorCount:               5,
				shouldPoolOperationInProgress: true,
				loopDelay:                     time.Microsecond * 100,
				poolInfo: &zpool.PoolMocker{
					TestConfig: zpool.TestConfig{
						ZpoolCommand: zpool.ZpoolCommandError{
							ZpoolAddError: true,
						},
					},
				},
				isDay2OperationNeedToPerform: true,
				dataRaidGroups: []cstor.RaidGroup{
					{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-58"},
						{BlockDeviceName: "blockdevice-59"}},
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
			f.openebsClient.CstorV1().CStorPoolInstances("openebs").Create(context.TODO(), test.cspi, metav1.CreateOptions{})
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
				Get(context.TODO(), test.cspi.Name, metav1.GetOptions{})
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
			if test.expectedPoolOperationPending {
				if ok, msg := f.isPoolOperationPending(testutil.GetKey(test.cspi, t), test.testConfig); !ok {
					t.Errorf("Expected pool operation to be in pending state but %s", msg)
				}
			}
		})
	}
	os.Unsetenv(string(common.OpenEBSIOPoolName))
	os.Unsetenv(util.Namespace)
}

// TestCSPIPoolReplacement will provision pool and then it
// will perform replacement operation by replacing blockdevices
func TestCSPIBlockDeviceReplacement(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	f.createFakeBlockDevices(25, "node1")
	f.fakeNodeCreator("node1")

	tests := map[string]struct {
		cspi                         *cstor.CStorPoolInstance
		shouldVerifyCSPIAutoGenerate bool
		shouldVerifyCSPIStatus       bool
		expectedPoolOperationPending bool
		testConfig                   *testConfig
	}{
		"Provision Mirror Pool And Replace BlockDevice": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-mirror-replace").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo-stripe-replace"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("mirror")).
				WithDataRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-1"},
							{BlockDeviceName: "blockdevice-2"},
						},
					},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-3"},
							{BlockDeviceName: "blockdevice-4"},
						},
					},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount: 4,
				loopDelay: time.Microsecond * 100,
				poolInfo: &zpool.PoolMocker{
					TestConfig: zpool.TestConfig{
						ResilveringProgress: 2,
					},
				},
				isDay2OperationNeedToPerform: true,
				replaceBlockDevices: map[string]string{
					"blockdevice-2": "blockdevice-5",
					"blockdevice-4": "blockdevice-6",
				},
			},
		},
		"Provision Raidz Pool And Replace Both Group BlockDevices": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-raidz-replace").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo-raidz-replace"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("raidz").
					WithWriteCacheGroupType("raidz")).
				WithDataRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-7"},
							{BlockDeviceName: "blockdevice-8"},
							{BlockDeviceName: "blockdevice-9"},
						},
					},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-10"},
							{BlockDeviceName: "blockdevice-11"},
							{BlockDeviceName: "blockdevice-12"},
						},
					},
				}).
				WithWriteCacheRaidGroups([]cstor.RaidGroup{
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-13"},
							{BlockDeviceName: "blockdevice-14"},
							{BlockDeviceName: "blockdevice-15"},
						},
					},
					{
						CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
							{BlockDeviceName: "blockdevice-16"},
							{BlockDeviceName: "blockdevice-17"},
							{BlockDeviceName: "blockdevice-18"},
						},
					},
				}).
				WithNewVersion(version.GetVersion()),
			shouldVerifyCSPIAutoGenerate: true,
			shouldVerifyCSPIStatus:       true,
			testConfig: &testConfig{
				loopCount: 4,
				loopDelay: time.Microsecond * 100,
				poolInfo: &zpool.PoolMocker{
					TestConfig: zpool.TestConfig{
						ResilveringProgress: 2,
					},
				},
				isDay2OperationNeedToPerform: true,
				replaceBlockDevices: map[string]string{
					"blockdevice-9":  "blockdevice-19",
					"blockdevice-12": "blockdevice-20",
					"blockdevice-14": "blockdevice-21",
					"blockdevice-15": "blockdevice-22",
					"blockdevice-16": "blockdevice-23",
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
			f.openebsClient.CstorV1().CStorPoolInstances("openebs").Create(context.TODO(), test.cspi, metav1.CreateOptions{})
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
				Get(context.TODO(), test.cspi.Name, metav1.GetOptions{})
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
			if test.expectedPoolOperationPending {
				if ok, msg := f.isPoolOperationPending(testutil.GetKey(test.cspi, t), test.testConfig); !ok {
					t.Errorf("Expected pool operation to be in pending state but %s", msg)
				}
			} else {
				if ok, msg := f.isReplacementMarksExists(*test.testConfig); ok {
					t.Errorf("Expected not to have any replacement marks but %s", msg)
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
   |                                   **Test CSPI Status**                                                     |
   |                                                                                                            |
   |                                                                                                            |
   -------------------------------------------------------------------------------------------------------------
*/

func TestCSPIStatus(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	f.createFakeBlockDevices(25, "node1")
	f.fakeNodeCreator("node1")
	tests := map[string]struct {
		cspi                  *cstor.CStorPoolInstance
		testConfig            *testConfig
		isExpectedEmptyStatus bool
		isReplicaInfoExpected bool
	}{
		"Provision Stripe Pool and Check CSPI status": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-stripe").
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
			testConfig: &testConfig{
				loopCount: 4,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
				volumeInfo: &zfs.VolumeMocker{
					TestConfig: zfs.TestConfig{
						HealthyReplicas:     10,
						ProvisionedReplicas: 5,
					},
				},
			},
			isReplicaInfoExpected: true,
		},
		"Provision Stripe Pool and return error for zfs list": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-bar-stripe").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-bar-stripe"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("stripe")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-2"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			testConfig: &testConfig{
				loopCount: 4,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
				volumeInfo: &zfs.VolumeMocker{
					TestConfig: zfs.TestConfig{
						HealthyReplicas:     1,
						ProvisionedReplicas: 2,
						ZFSCommand: zfs.ZFSCommandError{
							ZFSListError: true,
						},
					},
				},
				ejectErrorCount: 5,
			},
			isReplicaInfoExpected: false,
		},
		"Provision Stripe Pool and return error for zfs stats": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-1").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo1"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("stripe")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-3"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			testConfig: &testConfig{
				loopCount: 4,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
				volumeInfo: &zfs.VolumeMocker{
					TestConfig: zfs.TestConfig{
						HealthyReplicas:     1,
						ProvisionedReplicas: 2,
						ZFSCommand: zfs.ZFSCommandError{
							ZFSStatsError: true,
						},
					},
				},
				ejectErrorCount: 5,
			},
			isReplicaInfoExpected: false,
		},
		"Provision Stripe Pool and eject the fake error in zfs list at 3 iteration": {
			cspi: cstor.NewCStorPoolInstance().
				WithName("cspi-foo-2").
				WithNamespace("openebs").
				WithLabels(map[string]string{types.CStorPoolClusterLabelKey: "cspc-foo2"}).
				WithNodeName("node1").
				WithPoolConfig(*cstor.NewPoolConfig().
					WithDataRaidGroupType("stripe")).
				WithDataRaidGroups([]cstor.RaidGroup{{
					CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
						{BlockDeviceName: "blockdevice-4"},
					},
				},
				}).
				WithNewVersion(version.GetVersion()),
			testConfig: &testConfig{
				loopCount: 4,
				loopDelay: time.Microsecond * 100,
				poolInfo:  &zpool.PoolMocker{},
				volumeInfo: &zfs.VolumeMocker{
					TestConfig: zfs.TestConfig{
						HealthyReplicas:     1,
						ProvisionedReplicas: 2,
						ZFSCommand: zfs.ZFSCommandError{
							ZFSStatsError: true,
						},
					},
				},
				// Ejecting error at 3rd reconciliation
				ejectErrorCount: 3,
			},
			isReplicaInfoExpected: true,
		},
	}

	os.Setenv(string(common.OpenEBSIOPoolName), "1234")
	os.Setenv(util.Namespace, "openebs")
	common.Init()
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			provisionedReplicas :=
				test.testConfig.volumeInfo.TestConfig.ProvisionedReplicas +
					test.testConfig.volumeInfo.TestConfig.HealthyReplicas
			healthyReplicas := test.testConfig.volumeInfo.TestConfig.HealthyReplicas
			test.cspi.Kind = "CStorPoolInstance"
			// Create a CSPI to persist it in a fake store
			f.openebsClient.CstorV1().CStorPoolInstances("openebs").Create(context.TODO(), test.cspi, metav1.CreateOptions{})
			// Create claims for blockdevices exist on cspi
			err := f.prepareCSPIForDeploying(test.cspi)
			if err != nil {
				t.Errorf("Test: %q failed to prepare pools %s", name, err.Error())
			}
			f.run_(testutil.GetKey(test.cspi, t), true, false, test.testConfig)

			// CSPI controller is to create pools and manage it using zpool/zfs
			// command line utility. Since there is no real zrepl process is
			// running we can cspi status
			cspi, err := f.openebsClient.
				CstorV1().
				CStorPoolInstances(test.cspi.Namespace).
				Get(context.TODO(), test.cspi.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("error getting cspc %s: %v", cspi.Name, err)
			}
			if !test.isExpectedEmptyStatus {
				err = f.verifyCSPIStatus(cspi, test.testConfig)
				if err != nil {
					t.Errorf("Test: %q validation failed %s", name, err.Error())
				}
				if test.isReplicaInfoExpected {
					if cspi.Status.ProvisionedReplicas != int32(provisionedReplicas) {
						t.Errorf("CSPI %s expected to have %d provisioned replicas but got %d",
							name,
							provisionedReplicas,
							cspi.Status.ProvisionedReplicas,
						)
					}
					if cspi.Status.HealthyReplicas != int32(healthyReplicas) {
						t.Errorf("CSPI %s expected to have %d healthy replicas but got %d",
							name,
							healthyReplicas,
							cspi.Status.HealthyReplicas,
						)
					}
				} else {
					if cspi.Status.ProvisionedReplicas != int32(0) {
						t.Errorf("CSPI %s expected to have %d provisioned replicas but got %d",
							name,
							0,
							cspi.Status.ProvisionedReplicas,
						)
					}
					if cspi.Status.HealthyReplicas != int32(0) {
						t.Errorf("CSPI %s expected to have %d healthy replicas but got %d",
							name,
							0,
							cspi.Status.HealthyReplicas,
						)
					}
				}
			}

		})
	}
	os.Unsetenv(string(common.OpenEBSIOPoolName))
	os.Unsetenv(util.Namespace)
}

func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if action.Matches("create", "cstorpoolinstances") {
			continue
		}
		ret = append(ret, action)
	}
	return ret
}

// TestRun will test whether it is able to handle signal correctly
func TestRun(t *testing.T) {
	tests := map[string]struct {
		cspiList            *cstor.CStorPoolInstanceList
		updatedCSPIName     string
		stopChan            chan struct{}
		expectedActionCount int
	}{
		"When corresponding pool-manager CSPI exist in the system": {
			cspiList: &cstor.CStorPoolInstanceList{
				Items: []cstor.CStorPoolInstance{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-1-cspi-1",
							Namespace: "openebs",
							UID:       k8stypes.UID("1234"),
						},
						Status: cstor.CStorPoolInstanceStatus{
							Phase: "OFFLINE",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-1-cspi-2",
							Namespace: "openebs",
							UID:       k8stypes.UID("123456"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-1-cspi-3",
							Namespace: "openebs",
							UID:       k8stypes.UID("ADSA123456"),
						},
					},
				},
			},
			updatedCSPIName:     "test-1-cspi-1",
			stopChan:            make(chan struct{}),
			expectedActionCount: 2,
		},
		"When corresponding pool-manager CSPI doesn't exist in the system": {
			cspiList: &cstor.CStorPoolInstanceList{
				Items: []cstor.CStorPoolInstance{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-2-cspi-1",
							Namespace: "openebs",
							UID:       k8stypes.UID("1234ABC"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-2-cspi-2",
							Namespace: "openebs",
							UID:       k8stypes.UID("123456"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-2-cspi-3",
							Namespace: "openebs",
							UID:       k8stypes.UID("ADSA123456"),
						},
					},
				},
			},
			updatedCSPIName:     "",
			stopChan:            make(chan struct{}),
			expectedActionCount: 1,
		},
		"When pool is not created but pool-manager restarted": {
			cspiList: &cstor.CStorPoolInstanceList{
				Items: []cstor.CStorPoolInstance{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-3-cspi-1",
							Namespace: "openebs",
							UID:       k8stypes.UID("1234"),
						},
						Status: cstor.CStorPoolInstanceStatus{
							Phase: "",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-3-cspi-2",
							Namespace: "openebs",
							UID:       k8stypes.UID("1234ABC"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-3-cspi-3",
							Namespace: "openebs",
							UID:       k8stypes.UID("123456"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-3-cspi-4",
							Namespace: "openebs",
							UID:       k8stypes.UID("ADSA123456"),
						},
					},
				},
			},
			updatedCSPIName:     "",
			stopChan:            make(chan struct{}),
			expectedActionCount: 1,
		},
		"When users are trying to recreate a pool manually": {
			cspiList: &cstor.CStorPoolInstanceList{
				Items: []cstor.CStorPoolInstance{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-4-cspi-1",
							Namespace: "openebs",
							UID:       k8stypes.UID("1234"),
						},
						Status: cstor.CStorPoolInstanceStatus{
							Phase: cstor.CStorPoolStatusPending,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-4-cspi-2",
							Namespace: "openebs",
							UID:       k8stypes.UID("1234ABC"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-4-cspi-3",
							Namespace: "openebs",
							UID:       k8stypes.UID("123456"),
						},
					},
				},
			},
			updatedCSPIName:     "",
			stopChan:            make(chan struct{}),
			expectedActionCount: 1,
		},
	}
	os.Setenv(string(common.OpenEBSIOCSPIID), "1234")
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			// Creating clientset for each test to verify no.of etcd calls are going
			f := newFixture(t)
			f.SetFakeClient()
			testConfig := &testConfig{}
			c, _, err := f.newCSPIController(testConfig)
			if err != nil {
				t.Fatalf("Failed to instantiate fake controller")
			}

			// Create fake objects in etcd
			for _, cspiObj := range test.cspiList.Items {
				_, _ = c.clientset.CstorV1().CStorPoolInstances(cspiObj.Namespace).Create(context.TODO(), &cspiObj, metav1.CreateOptions{})
			}

			done := make(chan bool)
			go func(chan bool) {
				// close the channel so that Run will return
				close(test.stopChan)
				c.Run(1, test.stopChan)
				done <- true
			}(done)
			// Waiting for run to complete
			<-done

			actions := filterInformerActions(f.openebsClient.Actions())
			if len(actions) != test.expectedActionCount {
				t.Errorf("expected %d count of actions but got %d", test.expectedActionCount, len(actions))
			}

			for _, cspiObj := range test.cspiList.Items {
				updatedCSPIObj, _ := c.clientset.CstorV1().
					CStorPoolInstances(cspiObj.Namespace).
					Get(context.TODO(), cspiObj.Name, metav1.GetOptions{})
				if updatedCSPIObj.Name == test.updatedCSPIName &&
					updatedCSPIObj.Status.Phase != cstor.CStorPoolStatusOffline {
					t.Errorf("expected CSPI %s status to be updated to %s but got %s",
						updatedCSPIObj.Name,
						cstor.CStorPoolStatusOffline,
						updatedCSPIObj.Status.Phase,
					)
				}
			}
		})
	}
	os.Unsetenv(string(common.OpenEBSIOCSPIID))
}
