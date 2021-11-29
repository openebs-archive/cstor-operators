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
	zfs "github.com/openebs/cstor-operators/pkg/zcmd"
	"k8s.io/klog"
)

// Delete will destroy the pool for given cspi.
// It will also perform labelclear for pool disk.
func (oc *OperationsConfig) Delete(cspi *cstor.CStorPoolInstance) error {
	zpoolName := PoolName()
	klog.Infof("Destroying a pool {%s}", zpoolName)

	// Let's check if pool exists or not
	if poolExist := checkIfPoolPresent(zpoolName, oc.zcmdExecutor); !poolExist {
		klog.Infof("Pool %s not imported.. so, can't destroy", zpoolName)
		return nil
	}

	// First delete a pool
	ret, err := zfs.NewPoolDestroy().
		WithPool(zpoolName).
		WithExecutor(oc.zcmdExecutor).
		Execute()
	if err != nil {
		klog.Errorf("Failed to destroy a pool {%s}.. %s", ret, err.Error())
		return err
	}

	// We successfully deleted the pool.
	// We also need to clear the label for attached disk
	oc.ClearPoolLabel(cspi.GetAllRaidGroups()...)

	return nil
}

// ClearPoolLabel clears the pool labels on disks
func (oc *OperationsConfig) ClearPoolLabel(raidGroups ...cstor.RaidGroup) {
	for _, r := range raidGroups {
		disklist, err := oc.getPathForBdevList(r.CStorPoolInstanceBlockDevices)
		if err != nil {
			klog.Errorf("Failed to fetch vdev path, skipping labelclear.. %s", err.Error())
		}
		for _, v := range disklist {
			if _, err := zfs.NewPoolLabelClear().
				WithForceFully(true).
				WithVdev(v[0]).
				WithExecutor(oc.zcmdExecutor).
				Execute(); err != nil {
				klog.Errorf("Failed to perform label clear for disk {%s}.. %s", v, err.Error())
			}
		}
	}
}
