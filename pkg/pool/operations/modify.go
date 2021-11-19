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
	"github.com/openebs/api/v3/pkg/apis/types"
	cspiutil "github.com/openebs/cstor-operators/pkg/controllers/cspi-controller/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DeviceTypeSpare .. spare device type
	DeviceTypeSpare = "spare"
	// DeviceTypeReadCache .. read cache device type
	DeviceTypeReadCache = "cache"
	// DeviceTypeWriteCache .. write cache device type
	DeviceTypeWriteCache = "log"
	// DeviceTypeData .. data disk device type
	DeviceTypeData = "data"
)

//TODO: Get better naming conventions
type raidConfiguration struct {
	RaidGroupType string
	RaidGroups    []cstor.RaidGroup
}

func getRaidGroupsConfigMap(cspi *cstor.CStorPoolInstance) map[string]raidConfiguration {
	raidGroupsMap := map[string]raidConfiguration{}
	raidGroupsMap[DeviceTypeData] = raidConfiguration{
		RaidGroups:    cspi.Spec.DataRaidGroups,
		RaidGroupType: cspi.Spec.PoolConfig.DataRaidGroupType,
	}
	raidGroupsMap[DeviceTypeWriteCache] = raidConfiguration{
		RaidGroups:    cspi.Spec.WriteCacheRaidGroups,
		RaidGroupType: cspi.Spec.PoolConfig.WriteCacheGroupType,
	}
	return raidGroupsMap
}

// Update will update the deployed pool according to given cspi object
// NOTE: Update returns both CSPI as well as error
func (oc *OperationsConfig) Update(cspi *cstor.CStorPoolInstance) (*cstor.CStorPoolInstance, error) {
	var isObjChanged, isRaidGroupChanged, isReplacementTriggered bool
	var replacingBlockDeviceCount int

	bdClaimList, err := oc.getBlockDeviceClaimList(
		types.CStorPoolClusterLabelKey,
		cspi.GetLabels()[types.CStorPoolClusterLabelKey])
	if err != nil {
		return cspi, err
	}

	raidGroupConfigMap := getRaidGroupsConfigMap(cspi)

	for _, raidGroupsConfig := range raidGroupConfigMap {
		// first we will check if there any bdev is replaced or removed
		for raidIndex := 0; raidIndex < len(raidGroupsConfig.RaidGroups); raidIndex++ {
			isRaidGroupChanged = false
			raidGroup := raidGroupsConfig.RaidGroups[raidIndex]

			for bdevIndex := 0; bdevIndex < len(raidGroup.CStorPoolInstanceBlockDevices); bdevIndex++ {
				bdev := raidGroup.CStorPoolInstanceBlockDevices[bdevIndex]

				bdClaim, er := bdClaimList.GetBlockDeviceClaimFromBDName(
					bdev.BlockDeviceName)
				if er != nil {
					// This case is not possible
					err = ErrorWrapf(err,
						"Failed to get claim of blockdevice {%s}.. %s",
						bdev.BlockDeviceName,
						er.Error())
					// If claim doesn't exist for current blockdevice continue with
					// other blockdevices in cspi
					continue
				}

				// If current blockdevice is replaced blockdevice then get the
				// predecessor from claim of current blockdevice and if current
				// blockdevice is not replaced then predecessorBDName will be empty
				predecessorBDName := bdClaim.GetAnnotations()[types.PredecessorBDLabelKey]
				oldPath := []string{}
				if predecessorBDName != "" {
					// Get device links from old block device
					oldPath, er = oc.getPathForBDev(predecessorBDName)
					if er != nil {
						err = ErrorWrapf(err, "Failed to check bdev change {%s}.. %s", bdev.BlockDeviceName, er.Error())
						continue
					}
					isReplacementTriggered = true
					replacingBlockDeviceCount += 1
				}

				diskPath := ""
				// Let's check if any replacement is needed for this BDev
				newPath, er := oc.getPathForBDev(bdev.BlockDeviceName)
				if er != nil {
					err = ErrorWrapf(err, "Failed to check bdev change {%s}.. %s", bdev.BlockDeviceName, er.Error())
				} else {
					if diskPath, er = oc.replacePoolVdev(cspi, oldPath, newPath); er != nil {
						err = ErrorWrapf(err, "Failed to replace bdev for {%s}.. %s", bdev.BlockDeviceName, er.Error())
						continue
					} else {
						if !IsEmpty(diskPath) && diskPath != bdev.DevLink {
							// Here We are updating in underlying slice so no problem
							// Let's update devLink with new path for this bdev
							raidGroup.CStorPoolInstanceBlockDevices[bdevIndex].DevLink = diskPath
							isRaidGroupChanged = true
						}
					}
				}
				// Only To Generate an BlockDevice Replacement event
				if len(oldPath) != 0 && len(newPath) != 0 {
					oc.recorder.Eventf(cspi,
						corev1.EventTypeNormal,
						"BlockDevice Replacement",
						"Replacement of %s BlockDevice with %s BlockDevice is in-Progress",
						predecessorBDName,
						bdev.BlockDeviceName,
					)
				}

				// If disk got replaced check resilvering status.
				// 1. If resilvering is in progress don't do any thing.
				// 2. If resilvering is completed then perform cleanup process
				//   2.1 Unclaim the old blockdevice which was used by pool
				//   2.2 Remove the annotation from blockdeviceclaim which is
				//       inuse by cstor pool
				if predecessorBDName != "" && !isResilveringInProgress(executeZpoolDump, cspi, diskPath, oc.zcmdExecutor) {
					oldBDClaim, _ := bdClaimList.GetBlockDeviceClaimFromBDName(
						predecessorBDName)
					if er := oc.cleanUpReplacementMarks(oldBDClaim, bdClaim); er != nil {
						err = ErrorWrapf(
							err,
							"Failed cleanup replacement marks of replaced blockdevice {%s}.. %s",
							bdev.BlockDeviceName,
							er.Error(),
						)
					} else {
						isReplacementTriggered = true
						replacingBlockDeviceCount -= 1
						oc.recorder.Eventf(cspi,
							corev1.EventTypeNormal,
							"BlockDevice Replacement",
							"Resilvering is successfull on BlockDevice %s",
							bdev.BlockDeviceName,
						)
					}
				}
			}
			// If raidGroup is changed then update the cspi.spec.raidgroup entry
			// If raidGroup doesn't have any blockdevice then remove that raidGroup
			// and set isObjChanged
			if isRaidGroupChanged {
				//NOTE: Remove below code since we are not supporting removal of raid group/block device alone
				if len(raidGroup.CStorPoolInstanceBlockDevices) == 0 {
					cspi.Spec.DataRaidGroups = append(cspi.Spec.DataRaidGroups[:raidIndex], cspi.Spec.DataRaidGroups[raidIndex+1:]...)
					// We removed the raidIndex entry cspi.Spec.raidGroup
					raidIndex--
				}
				isObjChanged = true
			}
		}
	}

	if isReplacementTriggered {
		if replacingBlockDeviceCount > 0 {
			// Add/Update BlockDevice Replacement condition In CSPI
			condition := cspiutil.NewCSPICondition(
				cstor.CSPIDiskReplacement,
				corev1.ConditionTrue,
				"BlockDeviceReplacementInprogress",
				fmt.Sprintf(
					"Resilvering %d no.of blockdevices... because of blockdevice replacement error: %v",
					replacingBlockDeviceCount, err),
			)
			cspiutil.SetCSPICondition(&cspi.Status, *condition)
		} else {
			// Update BlockDevice Replacement condition to false in CSPI
			condition := cspiutil.NewCSPICondition(cstor.CSPIDiskReplacement, corev1.ConditionFalse, "BlockDeviceReplacementSucceess", "Blockdevice replacement was successfully completed")
			cspiutil.SetCSPICondition(&cspi.Status, *condition)
		}
		isObjChanged = true
	}

	if isObjChanged {
		if ncspi, er := oc.openebsclientset.
			CstorV1().
			CStorPoolInstances(cspi.Namespace).
			Update(context.TODO(), cspi, metav1.UpdateOptions{}); er != nil {
			err = ErrorWrapf(err, "Failed to update object.. err {%s}", er.Error())
		} else {
			cspi = ncspi
		}
	}

	//TODO revisit for day 2 ops
	if ncspi, er := oc.updateNewVdevFromCSPI(cspi); er != nil {
		oc.recorder.Eventf(cspi,
			corev1.EventTypeWarning,
			"Pool Expansion",
			"Failed to expand pool... Error: %s", er.Error(),
		)
		err = ErrorWrapf(err, "Pool expansion... err {%s}", er.Error())
	} else {
		cspi = ncspi
	}

	return cspi, err
}
