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
	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	zfs "github.com/openebs/cstor-operators/pkg/zcmd"
	"github.com/pkg/errors"
	"k8s.io/klog"
)

const (
	// DeviceTypeSpare .. spare device type
	DeviceTypeSpare = "spare"
	// DeviceTypeReadCache .. read cache device type
	DeviceTypeReadCache = "cache"
	// DeviceTypeWriteCache .. write cache device type
	DeviceTypeWriteCache = "log"
	// DeviceTypeEmpty .. empty device type.. data disk
	DeviceTypeEmpty = ""
)

// addRaidGroup add given raidGroup to pool
func (oc *OperationsConfig)addRaidGroup(csp *cstor.CStorPoolInstance, r cstor.RaidGroup) error {
	var vdevlist []string

	ptype := csp.Spec.PoolConfig.DataRaidGroupType
	if len(ptype) == 0 {
		// type is not mentioned, return with error
		return errors.New("type for data raid group not found")
	}
    // ToDo: WriteChacheDataRaidGroupd ??
	// deviceType := getDeviceType(r)

	disklist, err := oc.getPathForBdevList(r.BlockDevices)
	if err != nil {
		klog.Errorf("Failed to get list of disk-path : %s", err.Error())
		return err
	}

	for _, v := range disklist {
		vdevlist = append(vdevlist, v[0])
	}

	_, err = zfs.NewPoolExpansion().
		//WithDeviceType(deviceType).
		WithType(ptype).
		WithPool(PoolName(csp)).
		WithVdevList(vdevlist).
		Execute()
	return err
}

// addNewVdevFromCSP will add new disk, which is not being used in pool, from csp to given pool
func (oc *OperationsConfig)addNewVdevFromCSP(csp *cstor.CStorPoolInstance) error {
	var err error

	poolTopology, err := zfs.NewPoolDump().
		WithPool(PoolName(csp)).
		WithStripVdevPath().
		Execute()
	if err != nil {
		return errors.Errorf("Failed to fetch pool topology.. %s", err.Error())
	}

	for _, raidGroup := range csp.Spec.DataRaidGroups {
		wholeGroup := true
		var devlist []string

		for _, bdev := range raidGroup.BlockDevices {
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
			if er := oc.addRaidGroup(csp, raidGroup); er != nil {
				err = ErrorWrapf(err, "Failed to add raidGroup{%#v}.. %s", raidGroup, er.Error())
			}
		} else if len(devlist) != 0 && csp.Spec.PoolConfig.DataRaidGroupType == string(cstor.PoolStriped) {
			// ToDo: WriteCacheRaidGroups
			if _, er := zfs.NewPoolExpansion().
				//WithDeviceType(getDeviceType(raidGroup)).
				WithVdevList(devlist).
				WithPool(PoolName(csp)).
				Execute(); er != nil {
				err = ErrorWrapf(err, "Failed to add devlist %v.. err {%s}", devlist, er.Error())
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
		WithPool(PoolName(cspi)).
		Execute()
	if err != nil {
		return "", errors.Errorf("Failed to fetch pool topology.. %s", err.Error())
	}

	if usedPath, isUsed = checkIfDeviceUsed(npath, poolTopology); isUsed {
		return usedPath, nil
	}

	if len(oldPaths) == 0 {
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
		WithPool(PoolName(cspi)).
		Execute()
	return npath[0], err
}
