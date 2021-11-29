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
	"strings"

	"github.com/openebs/api/v3/pkg/apis/types"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	zpool "github.com/openebs/api/v3/pkg/internalapis/apis/cstor"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/openebs/cstor-operators/pkg/pool"
	zcmd "github.com/openebs/cstor-operators/pkg/zcmd"
	bin "github.com/openebs/cstor-operators/pkg/zcmd/bin"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var SupportedCompressionTypes = map[string]bool{
	"on":     true,
	"off":    true,
	"lz4":    true,
	"gle":    true,
	"lzjb":   true,
	"gzip":   true,
	"gzip-1": true,
	"gzip-2": true,
	"gzip-3": true,
	"gzip-4": true,
	"gzip-5": true,
	"gzip-6": true,
	"gzip-7": true,
	"gzip-8": true,
	"gzip-9": true,
}

func (oc *OperationsConfig) getPathForBdevList(bdevs []cstor.CStorPoolInstanceBlockDevice) (map[string][]string, error) {
	var err error

	vdev := make(map[string][]string, len(bdevs))
	for _, b := range bdevs {
		path, er := oc.getPathForBDev(b.BlockDeviceName)
		if er != nil || len(path) == 0 {
			err = ErrorWrapf(err, "Failed to fetch path for bdev {%s} {%s}", b.BlockDeviceName, er.Error())
			continue
		}
		vdev[b.BlockDeviceName] = path
	}
	return vdev, err
}

func (oc *OperationsConfig) getPathForBDev(bdev string) ([]string, error) {
	var path []string
	// TODO: replace `NAMESPACE` with env variable from CSPI deployment
	bd, err := oc.openebsclientset.
		OpenebsV1alpha1().
		BlockDevices(util.GetEnv(util.Namespace)).
		Get(context.TODO(), bdev, metav1.GetOptions{})
	if err != nil {
		return path, err
	}
	return getPathForBDevFromBlockDevice(bd), nil
}

func getZFSDeviceType(dType string) string {
	if dType == DeviceTypeData {
		return ""
	}
	return dType
}

func getPathForBDevFromBlockDevice(bd *openebsapis.BlockDevice) []string {
	var paths []string
	if len(bd.Spec.DevLinks) != 0 {
		for _, v := range bd.Spec.DevLinks {
			paths = append(paths, v.Links...)
		}
	}

	if len(bd.Spec.Path) != 0 {
		paths = append(paths, bd.Spec.Path)
	}
	return paths
}

// checkIfPoolPresent returns true if pool is available for operations
func checkIfPoolPresent(name string, executor bin.Executor) bool {
	if _, err := zcmd.NewPoolGetProperty().
		WithParsableMode(true).
		WithScriptedMode(true).
		WithField("value").
		WithProperty("name").
		WithPool(name).
		WithExecutor(executor).
		Execute(); err != nil {
		return false
	}
	return true
}

/*
func isBdevPathChanged(bdev cstor.CStorPoolClusterBlockDevice) ([]string, bool, error) {
	var err error
	var isPathChanged bool

	newPath, er := getPathForBDev(bdev.BlockDeviceName)
	if er != nil {
		err = errors.Errorf("Failed to get bdev {%s} path err {%s}", bdev.BlockDeviceName, er.Error())
	}

	if err == nil && !util.ContainsString(newPath, bdev.DevLink) {
		isPathChanged = true
	}

	return newPath, isPathChanged, err
}
*/

func compareDisk(path []string, d []zpool.Vdev) (string, bool) {
	for _, v := range d {
		if util.ContainsString(path, v.Path) {
			return v.Path, true
		}
		for _, p := range v.Children {
			if util.ContainsString(path, p.Path) {
				return p.Path, true
			}
			if path, r := compareDisk(path, p.Children); r {
				return path, true
			}
		}
	}
	return "", false
}

func checkIfDeviceUsed(path []string, t zpool.Topology) (string, bool) {
	var isUsed bool
	var usedPath string

	if usedPath, isUsed = compareDisk(path, t.VdevTree.Topvdev); isUsed {
		return usedPath, isUsed
	}

	if usedPath, isUsed = compareDisk(path, t.VdevTree.Spares); isUsed {
		return usedPath, isUsed
	}

	if usedPath, isUsed = compareDisk(path, t.VdevTree.Readcache); isUsed {
		return usedPath, isUsed
	}
	return usedPath, isUsed
}

// checkIfPoolIsImportable checks if the pool is imported or not. If the pool
// is present on the disk but  not imported it returns true as the pool can be
// imported. It also returns false if pool is not found on the disk.
func (oc *OperationsConfig) checkIfPoolIsImportable(cspi *cstor.CStorPoolInstance) (string, bool, error) {
	var cmdOut []byte
	var err error

	bdPath, err := oc.getPathForBDev(cspi.Spec.DataRaidGroups[0].CStorPoolInstanceBlockDevices[0].BlockDeviceName)
	if err != nil {
		return "", false, err
	}

	devID := pool.GetDevPathIfNotSlashDev(bdPath[0])
	if len(devID) != 0 {
		cmdOut, err = zcmd.NewPoolImport().WithDirectory(devID).WithExecutor(oc.zcmdExecutor).Execute()
		if strings.Contains(string(cmdOut), PoolName()) {
			return string(cmdOut), true, nil
		}
	}
	// there are some cases when import is succesful but zpool command return
	// noisy errors, hence better to check contains before return error
	cmdOut, err = zcmd.NewPoolImport().WithExecutor(oc.zcmdExecutor).Execute()
	if strings.Contains(string(cmdOut), PoolName()) {
		return string(cmdOut), true, nil
	}
	return string(cmdOut), false, err
}

// getBlockDeviceClaimList returns list of block device claims based on the
// label passed to the function
func (oc *OperationsConfig) getBlockDeviceClaimList(key, value string) (
	*openebsapis.BlockDeviceClaimList, error) {
	namespace := util.GetEnv(util.Namespace)
	bdcClient := oc.openebsclientset.OpenebsV1alpha1().BlockDeviceClaims(namespace)
	bdcAPIList, err := bdcClient.List(context.TODO(), metav1.ListOptions{
		LabelSelector: key + "=" + value,
	})
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to list bdc related to key: %s value: %s",
			key,
			value,
		)
	}
	return bdcAPIList, nil
}

func executeZpoolDump(cspi *cstor.CStorPoolInstance, zcmdExecutor bin.Executor) (zpool.Topology, error) {
	return zcmd.NewPoolDump().
		WithPool(PoolName()).
		WithStripVdevPath().
		WithExecutor(zcmdExecutor).
		Execute()
}

// isResilveringInProgress returns true if resilvering is inprogress at cstor
// pool
func isResilveringInProgress(
	executeCommand func(cspi *cstor.CStorPoolInstance, executor bin.Executor) (zpool.Topology, error),
	cspi *cstor.CStorPoolInstance,
	path string,
	executor bin.Executor) bool {
	poolTopology, err := executeCommand(cspi, executor)
	if err != nil {
		// log error
		klog.Errorf("Failed to get pool topology error: %v", err)
		return true
	}
	vdev, isVdevExist := getVdevFromPath(path, poolTopology)
	if !isVdevExist {
		return true
	}
	// If device in raid group didn't got replaced then there won't be any info
	// related to scan stats
	if len(vdev.ScanStats) == 0 {
		return false
	}
	// If device didn't underwent resilvering then no.of scaned bytes will be
	// zero
	if vdev.VdevStats[zpool.VdevScanProcessedIndex] == 0 {
		return false
	}
	// To decide whether resilvering is completed then check following steps
	// 1. Current device should be child device.
	// 2. Device Scan State should be completed
	if len(vdev.Children) == 0 &&
		vdev.ScanStats[zpool.VdevScanStatsStateIndex] == uint64(zpool.PoolScanFinished) &&
		vdev.ScanStats[zpool.VdevScanStatsScanFuncIndex] == uint64(zpool.PoolScanFuncResilver) {
		return false
	}
	return true
}

func getVdevFromPath(path string, topology zpool.Topology) (zpool.Vdev, bool) {
	var vdev zpool.Vdev
	var isVdevExist bool

	if vdev, isVdevExist = zpool.
		VdevList(topology.VdevTree.Topvdev).
		GetVdevFromPath(path); isVdevExist {
		return vdev, isVdevExist
	}

	if vdev, isVdevExist = zpool.
		VdevList(topology.VdevTree.Spares).
		GetVdevFromPath(path); isVdevExist {
		return vdev, isVdevExist
	}

	if vdev, isVdevExist = zpool.
		VdevList(topology.VdevTree.Readcache).
		GetVdevFromPath(path); isVdevExist {
		return vdev, isVdevExist
	}
	return vdev, isVdevExist
}

//cleanUpReplacementMarks should be called only after resilvering is completed.
//It does the following work
// 1. RemoveFinalizer on old block device claim exists and delete the old block
//   device claim.
// 2. Remove link of old block device in new block device claim
// oldObj is block device claim of replaced block device object which is
// detached from pool
// newObj is block device claim of current block device object which is in use
// by pool
func (oc *OperationsConfig) cleanUpReplacementMarks(oldObj, newObj *openebsapis.BlockDeviceClaim) error {
	if oldObj != nil {
		if util.ContainsString(oldObj.Finalizers, types.CSPCFinalizer) {
			oldObj.RemoveFinalizer(types.CSPCFinalizer)
			_, err := oc.openebsclientset.OpenebsV1alpha1().BlockDeviceClaims(oldObj.Namespace).Update(context.TODO(), oldObj, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(
					err,
					"Failed to remove finalizer %s on claim %s of blockdevice %s",
					types.CSPCFinalizer,
					oldObj.Name,
					oldObj.Spec.BlockDeviceName,
				)
			}
		}
		err := oc.openebsclientset.OpenebsV1alpha1().BlockDeviceClaims(newObj.Namespace).Delete(context.TODO(), oldObj.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(
				err,
				"Failed to unclaim old blockdevice {%s}",
				oldObj.Spec.BlockDeviceName,
			)
		}
		klog.Infof("Triggered deletion on claim %s of blockdevice %s", oldObj.Name, oldObj.Spec.BlockDeviceName)
	}
	bdAnnotations := newObj.GetAnnotations()
	delete(bdAnnotations, types.PredecessorBDLabelKey)
	newObj.SetAnnotations(bdAnnotations)
	_, err := oc.openebsclientset.OpenebsV1alpha1().BlockDeviceClaims(newObj.Namespace).Update(context.TODO(), newObj, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(
			err,
			"Failed to remove annotation {%s} from blockdeviceclaim {%s}",
			types.PredecessorBDLabelKey,
			newObj.Name,
		)
	}
	klog.Infof("Cleared replacement marks on blockdevice %s", newObj.Name)
	return nil
}

// GetUnavailableDiskList returns the list of faulted disks from the current pool
func (oc *OperationsConfig) GetUnavailableDiskList(cspi *cstor.CStorPoolInstance) ([]string, error) {
	faultedDevices := []string{}
	topology, err := executeZpoolDump(cspi, oc.zcmdExecutor)
	if err != nil {
		return []string{}, errors.Wrapf(err, "failed to execute zpool dump")
	}
	raidGroupMap := getRaidGroupsConfigMap(cspi)
	for _, raidGroupsConfig := range raidGroupMap {
		for raidIndex := 0; raidIndex < len(raidGroupsConfig.RaidGroups); raidIndex++ {
			raidGroup := raidGroupsConfig.RaidGroups[raidIndex]
			for bdevIndex := 0; bdevIndex < len(raidGroup.CStorPoolInstanceBlockDevices); bdevIndex++ {
				bdev := raidGroup.CStorPoolInstanceBlockDevices[bdevIndex]
				if bdev.DevLink != "" {
					vdev, isPresent := getVdevFromPath(bdev.DevLink, topology)
					if !isPresent {
						klog.Errorf("BlockDevice %s doesn't exist in pool %s", bdev.BlockDeviceName, PoolName())
						continue
					}
					if vdev.VdevStats[zpool.VdevStateIndex] != uint64(zpool.VdevStateHealthy) {
						oc.recorder.Event(
							cspi,
							corev1.EventTypeWarning,
							"DeviceState",
							fmt.Sprintf("%s device was in %s state", bdev.BlockDeviceName, vdev.GetVdevState()))
						faultedDevices = append(faultedDevices, bdev.BlockDeviceName)
					}
				}
			}
		}
	}
	return faultedDevices, nil
}
