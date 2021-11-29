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

package v1alpha2

import (
	"fmt"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/cstor-operators/pkg/controllers/common"
	"github.com/openebs/cstor-operators/pkg/pool"
	"github.com/openebs/cstor-operators/pkg/volumereplica"
	zfs "github.com/openebs/cstor-operators/pkg/zcmd"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// Import will import pool for given CSPI object.
// It will always set `cachefile` property for that pool
// It will return following thing
// - If pool is imported or not
// - If any error occurred during import operation
func (oc *OperationsConfig) Import(cspi *cstor.CStorPoolInstance) (bool, error) {
	if poolExist := checkIfPoolPresent(PoolName(), oc.zcmdExecutor); poolExist {
		// If the pool is renamed and imported but the pool-mgmt restarts
		// for some reason then the annotation should be removed.
		delete(cspi.Annotations, types.OpenEBSCStorExistingPoolName)
		return true, nil
	}

	var poolImported, importable bool
	var err error
	// existingPoolName denotes the pool name that may be present
	// from previous version and needs to be imported with new name
	existingPoolName := cspi.Annotations[types.OpenEBSCStorExistingPoolName]
	if existingPoolName != "" {
		_, importable, err = oc.checkIfPoolIsImportable(cspi)
		if err != nil {
			return false, errors.Errorf("failed to verify if pool is importable: %s", err.Error())
		}
		if importable {
			// If the pool is renamed but not imported, remove the
			// annotation to avoid not found errors.
			existingPoolName = ""
			delete(cspi.Annotations, types.OpenEBSCStorExistingPoolName)
		}
	}

	// Pool is not imported.. Let's update the syncResource
	var cmdOut []byte
	common.SyncResources.IsImported = false

	bdPath, err := oc.getPathForBDev(cspi.Spec.DataRaidGroups[0].CStorPoolInstanceBlockDevices[0].BlockDeviceName)
	if err != nil {
		return false, err
	}

	klog.Infof("Importing pool %s %s", string(cspi.GetUID()), PoolName())
	devID := pool.GetDevPathIfNotSlashDev(bdPath[0])
	cacheFile := types.CStorPoolBasePath + types.CacheFileName

	if existingPoolName != "" {
		klog.Infof("Renaming pool %s to %s", existingPoolName, PoolName())
	}
	// Import the pool using cachefile
	// command will look like: zpool import -c <cachefile_path> -o <cachefile_path> <pool_name>
	// if existing pool name is present: zpool import -c <cachefile_path> -o <cachefile_path> <existing_pool_name> <pool_name>
	cmdOut, err = zfs.NewPoolImport().
		WithCachefile(cacheFile).
		WithProperty("cachefile", cacheFile).
		WithPool(existingPoolName).
		WithNewPool(PoolName()).
		WithExecutor(oc.zcmdExecutor).
		Execute()
	if err == nil {
		poolImported = true
	} else {
		// TODO may be possible that there is no pool exists or no cache file exists
		klog.Errorf("Failed to import pool by reading cache file: %s : %s", cmdOut, err.Error())
	}

	if !poolImported {
		// Import the pool without cachefile by scanning the directory
		// For sparse based pools import command: zpool import -d <parent_dir_sparse_files> -o <cachefile_path> <pool_name>
		// if existing pool name is present: zpool import -d <parent_dir_sparse_files> -o <cachefile_path> <existing_pool_name> <pool_name>
		// For device based pools import command: zpool import -o <cachefile_path> <pool_name>(by default it will scan /dev directory)
		// if existing pool name is present: zpool import -o <cachefile_path> <existing_pool_name> <pool_name>(by default it will scan /dev directory)
		cmdOut, err = zfs.NewPoolImport().
			WithDirectory(devID).
			WithProperty("cachefile", cacheFile).
			WithPool(existingPoolName).
			WithNewPool(PoolName()).
			WithExecutor(oc.zcmdExecutor).
			Execute()
	}

	if err != nil {
		// TODO may be possible that there is no pool exists..
		klog.Errorf("Failed to import pool by scanning directory: %s : %s", cmdOut, err.Error())
		return false, err
	}

	// after successful import of pool the annotation needs to be deleted
	// to avoid renaming of pool that is already renamed which will cause
	// pool not found errors
	delete(cspi.Annotations, types.OpenEBSCStorExistingPoolName)

	common.SyncResources.IsImported = true
	oc.recorder.Event(cspi,
		corev1.EventTypeNormal,
		"Pool "+string(common.SuccessImported),
		fmt.Sprintf("Pool Import successful: %v", PoolName()))
	return true, nil
}

// CheckImportedPoolVolume will notify CVR controller
// for new imported pool's volumes
func CheckImportedPoolVolume() {
	// ToDo: Fix this once cvr controller make in
	var err error

	if common.SyncResources.IsImported {
		return
	}

	// GetVolumes is called because, while importing a pool, volumes corresponding
	// to the pool are also imported. This needs to be handled and made visible
	// to cvr controller.
	common.InitialImportedPoolVol, err = volumereplica.GetVolumes()
	if err != nil {
		common.SyncResources.IsImported = false
		return
	}

	// make a check if initialImportedPoolVol is not empty, then notify cvr controller
	// through channel.
	if len(common.InitialImportedPoolVol) != 0 {
		common.SyncResources.IsImported = true
	} else {
		common.SyncResources.IsImported = false
	}
}
