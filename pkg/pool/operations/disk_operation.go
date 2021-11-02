/*
Copyright 2019 The OpenEBS Authors.

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

package v1alpha2

import (
	"context"
	"fmt"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	cspiutil "github.com/openebs/cstor-operators/pkg/controllers/cspi-controller/util"
	zfs "github.com/openebs/cstor-operators/pkg/zcmd"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// addRaidGroup add given raidGroup to pool
func (oc *OperationsConfig) addRaidGroup(r cstor.RaidGroup, dType, pType string) error {
	var vdevlist []string

	deviceType := getZFSDeviceType(dType)

	if len(pType) == 0 {
		// type is not mentioned, return with error
		return errors.Errorf("type for %s raid group not found", deviceType)
	}

	disklist, err := oc.getPathForBdevList(r.CStorPoolInstanceBlockDevices)
	if err != nil {
		klog.Errorf("Failed to get list of disk-path : %s", err.Error())
		return err
	}

	for _, v := range disklist {
		vdevlist = append(vdevlist, v[0])
	}

	_, err = zfs.NewPoolExpansion().
		WithDeviceType(deviceType).
		WithType(pType).
		WithPool(PoolName()).
		WithVdevList(vdevlist).
		WithExecutor(oc.zcmdExecutor).
		Execute()
	return err
}

// TODO: Get better naming convention from reviews
// updateNewVdevFromCSPI will add new disk, which is not being used in pool,
// from cspi to given pool. If there is any pool expansion process then below
// function will update the condition accordingly
func (oc *OperationsConfig) updateNewVdevFromCSPI(
	cspi *cstor.CStorPoolInstance) (*cstor.CStorPoolInstance, error) {
	var err error
	var isPoolExpansionTriggered, isStatusConditionChanged bool
	var newCondition *cstor.CStorPoolInstanceCondition
	successExpansionReason := "PoolExpansionSuccessful"

	poolTopology, err := zfs.NewPoolDump().
		WithPool(PoolName()).
		WithStripVdevPath().
		WithExecutor(oc.zcmdExecutor).
		Execute()
	if err != nil {
		return cspi, errors.Errorf("Failed to fetch pool topology.. %s", err.Error())
	}

	raidGroupConfigMap := getRaidGroupsConfigMap(cspi)

	for deviceType, raidGroupConfig := range raidGroupConfigMap {
		for _, raidGroup := range raidGroupConfig.RaidGroups {
			isRaidGroupExpanded := false
			wholeGroup := true
			var message string
			var devlist []string
			var newBlockDeviceList []string

			for _, bdev := range raidGroup.CStorPoolInstanceBlockDevices {
				newPath, er := oc.getPathForBDev(bdev.BlockDeviceName)
				if er != nil {
					return cspi, errors.Errorf("Failed get bdev {%s} path err {%s}", bdev.BlockDeviceName, er.Error())
				}
				if _, isUsed := checkIfDeviceUsed(newPath, poolTopology); !isUsed {
					devlist = append(devlist, newPath[0])
					newBlockDeviceList = append(newBlockDeviceList, bdev.BlockDeviceName)
				} else {
					wholeGroup = false
				}
			}
			/* Perform vertical Pool expansion only if entier raid group is added */
			if wholeGroup {
				isPoolExpansionTriggered = true
				if er := oc.addRaidGroup(raidGroup, deviceType, raidGroupConfig.RaidGroupType); er != nil {
					err = ErrorWrapf(err, "Failed to add raidGroup{%#v}.. %s", raidGroup, er.Error())
				} else {
					isRaidGroupExpanded = true
					message = fmt.Sprintf(
						"Pool Expanded Successfully By Adding RaidGroup With BlockDevices: %v device type: %s pool type: %s",
						raidGroup.GetBlockDevices(),
						deviceType,
						raidGroupConfig.RaidGroupType,
					)
				}
			} else if len(devlist) != 0 && raidGroupConfig.RaidGroupType == string(cstor.PoolStriped) {
				isPoolExpansionTriggered = true
				if ret, er := zfs.NewPoolExpansion().
					WithDeviceType(getZFSDeviceType(deviceType)).
					WithVdevList(devlist).
					WithPool(PoolName()).
					WithExecutor(oc.zcmdExecutor).
					Execute(); er != nil {
					err = ErrorWrapf(err, "Failed to add devlist %v.. err {%s} {%s}", devlist, string(ret), er.Error())
				} else {
					isRaidGroupExpanded = true
					message = fmt.Sprintf(
						"Pool Expanded Successfully By Adding BlockDevices: %v device type: %s pool type: %s",
						newBlockDeviceList,
						deviceType,
						raidGroupConfig.RaidGroupType,
					)
				}
			}
			if isRaidGroupExpanded {
				oc.recorder.Event(cspi, corev1.EventTypeNormal, "Pool Expansion", message)
				isPoolExpansionTriggered = true
			}
		}
	}

	// If expansion is triggered in nth reconciliation then in same reconciliation
	// expansion inprogress condition will be addede and in next subsequent
	// reconciliation if there are are no pending in expansion operation then status
	// will be updated to success
	condition := cspiutil.GetCSPICondition(cspi.Status, cstor.CSPIPoolExpansion)
	// If expansion is successfull we need to update the expansion condition as success
	if condition != nil && !isPoolExpansionTriggered && condition.Reason != successExpansionReason {
		newCondition = cspiutil.NewCSPICondition(
			cstor.CSPIPoolExpansion,
			corev1.ConditionFalse,
			successExpansionReason,
			"Pool expansion was successfull by adding blockdevices/raid groups",
		)
		isStatusConditionChanged = true
	} else if isPoolExpansionTriggered {
		newCondition = cspiutil.NewCSPICondition(
			cstor.CSPIPoolExpansion,
			corev1.ConditionTrue,
			"PoolExpansionInProgress",
			fmt.Sprintf("Pool expansion is in progress because of blockdevice/raid group addition error: %v", err),
		)
		isStatusConditionChanged = true
	}

	// When there is change in the condition then update condition into etcd
	if isStatusConditionChanged {
		cspiCopy := cspi.DeepCopy()
		cspiutil.SetCSPICondition(&cspi.Status, *newCondition)
		updatedCSPI, updateErr := oc.openebsclientset.
			CstorV1().
			CStorPoolInstances(cspi.Namespace).
			Update(context.TODO(), cspi, metav1.UpdateOptions{})
		if updateErr != nil {
			return cspiCopy, errors.Wrapf(
				updateErr,
				"failed to update cspi pool expansion conditions error: %v", err)
		}
		cspi = updatedCSPI
	}
	return cspi, err
}

/*
func removePoolVdev(csp *cstor.CStorPoolInstance, bdev cstor.CStorPoolClusterBlockDevice) error {
	if _, err := zfs.NewPoolRemove().
		WithDevice(bdev.DevLink).
		WithPool(PoolName(csp)).
		Execute(); err != nil {
		return err
	}

	// Let's clear the label for removed disk
	if _, err := zfs.NewPoolLabelClear().
		WithForceFully(true).
		WithVdev(bdev.DevLink).
		Execute(); err != nil {
		// Let's just log the error
		klog.Errorf("Failed to perform label clear for disk {%s}", bdev.DevLink)
	}

	return nil
}
*/

// replacePoolVdev will replace the given bdev disk with
// disk(i.e npath[0]) and return updated disk path(i.e npath[0])
//
// Note, if a new disk is already being used then we will
// not perform disk replacement and function will return
// the used disk path from given path(npath[])
func (oc *OperationsConfig) replacePoolVdev(cspi *cstor.CStorPoolInstance, oldPaths, npath []string) (string, error) {
	var usedPath string
	var isUsed bool
	if len(npath) == 0 {
		return "", errors.Errorf("Empty path for vdev")
	}

	// Wait! Device path may got changed due to import
	// Let's check if a device, having path `npath`, is already present in pool
	poolTopology, err := zfs.
		NewPoolDump().
		WithStripVdevPath().
		WithPool(PoolName()).
		WithExecutor(oc.zcmdExecutor).
		Execute()
	if err != nil {
		return "", errors.Errorf("Failed to fetch pool topology.. %s", err.Error())
	}

	if usedPath, isUsed = checkIfDeviceUsed(npath, poolTopology); isUsed {
		return usedPath, nil
	}

	if len(oldPaths) == 0 {
		// Might be pool expansion case i.e by adding new vdev/blockdevice
		return "", nil
	}

	// Device path may got changed after imports. So let's get the path used by
	// pool and trigger replace
	if usedPath, isUsed = checkIfDeviceUsed(oldPaths, poolTopology); !isUsed {
		// Might be a case where paths in the old blockdevice are not up to date
		return "", errors.Errorf("Old device links are not in use by pool")
	}

	// Replace the disk
	_, err = zfs.NewPoolDiskReplace().
		WithOldVdev(usedPath).
		WithNewVdev(npath[0]).
		WithPool(PoolName()).
		WithExecutor(oc.zcmdExecutor).
		Execute()
	if err == nil {
		klog.Infof("Triggered replacement of %s with %s on pool %s",
			usedPath,
			npath[0],
			PoolName(),
		)
	}
	return npath[0], err
}
