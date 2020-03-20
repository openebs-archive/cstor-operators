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
	"fmt"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	zfs "github.com/openebs/cstor-operators/pkg/zcmd"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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
		Execute()
	return err
}

// addNewVdevFromCSP will add new disk, which is not being used in pool, from cspi to given pool
func (oc *OperationsConfig) addNewVdevFromCSP(cspi *cstor.CStorPoolInstance) error {
	var err error

	poolTopology, err := zfs.NewPoolDump().
		WithPool(PoolName()).
		WithStripVdevPath().
		Execute()
	if err != nil {
		return errors.Errorf("Failed to fetch pool topology.. %s", err.Error())
	}

	raidGroupConfigMap := getRaidGroupsConfigMap(cspi)

	for deviceType, raidGroupConfig := range raidGroupConfigMap {
		for _, raidGroup := range raidGroupConfig.RaidGroups {
			isPoolExpanded := false
			wholeGroup := true
			var message string
			var devlist []string

			for _, bdev := range raidGroup.CStorPoolInstanceBlockDevices {
				newPath, er := oc.getPathForBDev(bdev.BlockDeviceName)
				if er != nil {
					return errors.Errorf("Failed get bdev {%s} path err {%s}", bdev.BlockDeviceName, er.Error())
				}
				if _, isUsed := checkIfDeviceUsed(newPath, poolTopology); !isUsed {
					devlist = append(devlist, newPath[0])
				} else {
					wholeGroup = false
				}
			}
			/* Perform vertical Pool expansion only if entier raid group is added */
			if wholeGroup {
				if er := oc.addRaidGroup(raidGroup, deviceType, raidGroupConfig.RaidGroupType); er != nil {
					err = ErrorWrapf(err, "Failed to add raidGroup{%#v}.. %s", raidGroup, er.Error())
				} else {
					isPoolExpanded = true
					message = fmt.Sprintf(
						"Pool Expanded Successfully By Adding RaidGroup With BlockDevices: %v device type: %s pool type: %s",
						raidGroup.GetBlockDevices(),
						deviceType,
						raidGroupConfig.RaidGroupType,
					)
				}
			} else if len(devlist) != 0 && raidGroupConfig.RaidGroupType == string(cstor.PoolStriped) {
				if _, er := zfs.NewPoolExpansion().
					WithDeviceType(getZFSDeviceType(deviceType)).
					WithVdevList(devlist).
					WithPool(PoolName()).
					Execute(); er != nil {
					err = ErrorWrapf(err, "Failed to add devlist %v.. err {%s}", devlist, er.Error())
				} else {
					isPoolExpanded = true
					message = fmt.Sprintf(
						"Pool Expanded Successfully By Adding BlockDevice Under Raid Group")
				}
			}
			if isPoolExpanded {
				oc.recorder.Event(cspi, corev1.EventTypeNormal, "Pool Expansion", message)
			}
		}
	}
	return err
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
func replacePoolVdev(cspi *cstor.CStorPoolInstance, oldPaths, npath []string) (string, error) {
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
		Execute()
	if err != nil {
		return "", errors.Errorf("Failed to fetch pool topology.. %s", err.Error())
	}

	if usedPath, isUsed = checkIfDeviceUsed(npath, poolTopology); isUsed {
		return usedPath, nil
	}

	if len(oldPaths) == 0 {
		// Might be pool expansion case i.e added new vdev
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
