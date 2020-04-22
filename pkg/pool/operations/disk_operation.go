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
	"github.com/openebs/cstor-operators/pkg/partition/partprobe"
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

	raidGroupConfigMap := getRaidGroupsConfiguration(cspi)

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
// and capacity
//
// NOTE: If new disk is already being used then we will
// not perform disk replacement operation and this function wil
// return the used disk path from given path(npath[]) and capacity
func replacePoolVdev(cspi *cstor.CStorPoolInstance, oldPaths, npath []string) (string, uint64, error) {
	var usedPath string
	var diskCapacity uint64
	var isUsed bool
	if len(npath) == 0 {
		return "", diskCapacity, errors.Errorf("Empty path for vdev")
	}

	// Wait! Device path may got changed due to import
	// Let's check if a device, having path `npath`, is already present in pool
	poolTopology, err := zfs.
		NewPoolDump().
		WithStripVdevPath().
		WithPool(PoolName()).
		Execute()
	if err != nil {
		return "", diskCapacity, errors.Errorf("Failed to fetch pool topology.. %s", err.Error())
	}

	if usedPath, isUsed = checkIfDeviceUsed(npath, poolTopology); isUsed {
		// If path exist in topology then vdev also will exist
		vdev, _ := getVdevFromPath(usedPath, poolTopology)
		return usedPath, vdev.Capacity, nil
	}

	if len(oldPaths) == 0 {
		// Might be pool expansion case i.e added new vdev/blockdevice
		return "", diskCapacity, nil
	}

	// Device path may got changed after imports. So let's get the path used by
	// pool and trigger replace
	if usedPath, isUsed = checkIfDeviceUsed(oldPaths, poolTopology); !isUsed {
		// Might be a case where paths in the old blockdevice are not up to date
		return "", diskCapacity, errors.Errorf("Old device links are not in use by pool")
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
		// Replaced deivce will exist in poolTopology after fetching
		// from zfs
		newPoolTopology, err := executeZpoolDump()
		if err != nil {
			klog.Warningf("failed to execute zpool dump after replacing the blockdevice error: %v", err)
			return npath[0], diskCapacity, nil
		}
		vdev, _ := getVdevFromPath(npath[0], newPoolTopology)
		return npath[0], vdev.Capacity, nil
	}
	return "", diskCapacity, err
}

// expandPool will perform pool expansion when underlying disk itself expanded.
// It perfrom following steps:
// 1. Trigger `zpool online -e <pool_name> <path_to_disk>` to recover from GPT PMBR mismatch error.
// =============== Error reported due to mismatch in kernel cache partition and disk =====
// ||    GPT PMBR size mismatch (1310719 != 1835007) will be corrected by w(rite).      ||
// ||    Disk /dev/sdb: 7 GiB, 7516192768 bytes, 1835008 sectors                        ||
// ||    Units: sectors of 1 * 4096 = 4096 bytes                                        ||
// ||    Sector size (logical/physical): 4096 bytes / 4096 bytes                        ||
// ||    I/O size (minimum/optimal): 32768 bytes / 1048576 bytes                        ||
// ||    Disklabel type: gpt                                                            ||
// ||    Disk identifier: FC9B07E5-02D0-394C-B5F8-EB25FF3C2E36                          ||
// ||                                                                                   ||
// ||    Device       Start     End Sectors  Size Type                                  ||
// ||    /dev/sdb1     2048 1292287 1290240  4.9G Solaris /usr & Apple ZFS              ||
// ||    /dev/sdb9  1292288 1308671   16384   64M Solaris reserved 1                    ||
// =======================================================================================
// 2. partprobe <path_to_disk> (To reload the partition into kernel cache).
// 3. Trigger `zpool online -e <pool_name> <path_to_disk>` to perfrom pool expansion.
func (oc *OperationsConfig) expandPool(path string, currentCapacity uint64) error {
	ret, err := executePoolExpansion(path)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to clear GPT PMBR size mismatch error via expansion of pool output: %s", string(ret))
	}

	ret, err = partprobe.NewDisk().
		WithDevice(path).
		Execute()
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to reload partitions by execute partprobe on device %s output: %s",
			path,
			string(ret))
	}

	ret, err = executePoolExpansion(path)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to execute pool expansion using disk %s output: %s", path, string(ret))
	}

	poolTopology, err := executeZpoolDump()
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to get pool dump",
		)
	}
	vdev, _ := getVdevFromPath(path, poolTopology)
	// since the currentCapacity is from zpool dump itself
	// but before expansion so good to compare
	if vdev.Capacity <= currentCapacity {
		return errors.Errorf("performed required steps to expand the pool but disk in pool not expanded")
	}

	return nil
}
