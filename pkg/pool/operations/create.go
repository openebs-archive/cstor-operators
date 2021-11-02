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
	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	zfs "github.com/openebs/cstor-operators/pkg/zcmd"
	"github.com/pkg/errors"
	"k8s.io/klog"
)

// Create will create the pool for given csp object
func (oc *OperationsConfig) Create(cspi *cstor.CStorPoolInstance) error {
	var err error

	// Let's check if there is any disk having the pool config
	// If so then we will not create the pool
	ret, notImported, err := oc.checkIfPoolIsImportable(cspi)
	if err != nil {
		return errors.Errorf("failed to verify if pool is importable: %s", err.Error())
	}
	if notImported {
		return errors.Errorf("Pool {%s} is in faulty state.. %s", PoolName(), ret)
	}

	klog.Infof("Creating a pool for %s %s", cspi.Name, PoolName())

	// First create a pool
	// TODO, IsWriteCache, IsSpare, IsReadCache should be disable for actual pool?

	// Lets say we need to execute following command
	// -- zpool create newpool mirror v0 v1 mirror v2 v3 log mirror v4 v5
	// Above command we will execute using following steps:
	// 1. zpool create newpool mirror v0 v1
	// 2. zpool add newpool log mirror v4 v5
	// 3. zpool add newpool mirror v2 v3
	cspiCopy := cspi.DeepCopy()
	for i, r := range cspiCopy.Spec.DataRaidGroups {
		// we found the main raidgroup. let's create the pool
		err = oc.createPool(cspiCopy, r)
		if err != nil {
			return errors.Errorf("Failed to create pool {%s} : %s",
				PoolName(), err.Error())
		}
		// Remove this raidGroup
		cspiCopy.Spec.DataRaidGroups = append(cspiCopy.Spec.DataRaidGroups[:i], cspiCopy.Spec.DataRaidGroups[i+1:]...)
		break
	}

	// We created the pool
	// Lets update it with extra config, if provided
	raidGroupConfigMap := getRaidGroupsConfigMap(cspiCopy)
	for deviceType, raidGroupConfig := range raidGroupConfigMap {
		for _, r := range raidGroupConfig.RaidGroups {
			if e := oc.addRaidGroup(r, deviceType, raidGroupConfig.RaidGroupType); e != nil {
				err = ErrorWrapf(err, "Failed to add raidGroup{%#v}.. %s", r, e.Error())
			}
		}
	}

	return err
}

func (oc *OperationsConfig) createPool(cspi *cstor.CStorPoolInstance, r cstor.RaidGroup) error {
	var vdevlist []string

	ptype := cspi.Spec.PoolConfig.DataRaidGroupType
	if len(ptype) == 0 {
		// type is not mentioned, return with error
		return errors.New("type for data raid group not found")
	}

	disklist, err := oc.getPathForBdevList(r.CStorPoolInstanceBlockDevices)
	if err != nil {
		return errors.Errorf("Failed to get list of disk-path : %s", err.Error())
	}

	for _, v := range disklist {
		vdevlist = append(vdevlist, v[0])
	}

	compressionType := cspi.Spec.PoolConfig.Compression
	if compressionType == "" {
		compressionType = "lz4"
	}

	ret, err := zfs.NewPoolCreate().
		WithType(ptype).
		WithProperty("cachefile", types.CStorPoolBasePath+types.CacheFileName).
		WithFSProperty("compression", compressionType).
		WithFSProperty("canmount", "off").
		WithPool(PoolName()).
		WithVdevList(vdevlist).
		WithExecutor(oc.zcmdExecutor).
		Execute()
	if err != nil {
		return errors.Errorf("Failed to create pool.. %s .. %s", string(ret), err.Error())
	}

	return nil
}
