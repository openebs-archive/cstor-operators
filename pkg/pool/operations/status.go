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
	"strings"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	zfs "github.com/openebs/cstor-operators/pkg/zcmd"
	bin "github.com/openebs/cstor-operators/pkg/zcmd/bin"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
)

// GetPropertyValue will return value of given property for given pool
func GetPropertyValue(poolName, property string, executor bin.Executor) (string, error) {
	ret, err := zfs.NewPoolGetProperty().
		WithScriptedMode(true).
		WithField("value").
		WithProperty(property).
		WithPool(poolName).
		WithExecutor(executor).
		Execute()
	if err != nil {
		return "", errors.Wrapf(err,
			"failed to get property %s value output: %s",
			property,
			string(ret),
		)
	}
	outStr := strings.Split(string(ret), "\n")
	return outStr[0], nil
}

// GetListOfPropertyValues will return value list for given property list
// NOTE: It will return the property values in the same order as property list
func (oc *OperationsConfig) GetListOfPropertyValues(
	poolName string, propertyList []string) ([]string, error) {
	ret, err := zfs.NewPoolGetProperty().
		WithScriptedMode(true).
		WithField("value").
		WithPropertyList(propertyList).
		WithPool(poolName).
		WithExecutor(oc.zcmdExecutor).
		Execute()
	if err != nil {
		return []string{}, err
	}
	// NOTE: Don't trim space there might be possibility for some
	// properties values might be empty. If we trim the space we
	// will lost the property values
	outStr := strings.Split(string(ret), "\n")
	return outStr, nil

}

// GetVolumePropertyValue is used to get pool properties using zfs commands
func (oc *OperationsConfig) GetVolumePropertyValue(poolName, property string) (string, error) {
	ret, err := zfs.NewVolumeGetProperty().
		WithScriptedMode(true).
		WithField("value").
		WithProperty(property).
		WithDataset(poolName).
		WithExecutor(oc.zcmdExecutor).
		Execute()
	if err != nil {
		return "", errors.Wrapf(err,
			"failed to get property %s value output: %s",
			property,
			string(ret),
		)
	}
	outStr := strings.Split(string(ret), "\n")
	return outStr[0], nil
}

// GetCSPICapacity returns the free, allocated and total capacities of pool in
// a structure
func (oc *OperationsConfig) GetCSPICapacity(poolName string) (cstor.CStorPoolInstanceCapacity, error) {
	propertyList := []string{"free", "allocated", "size"}
	cspiCapacity := cstor.CStorPoolInstanceCapacity{}
	valueList, err := oc.GetListOfPropertyValues(poolName, propertyList)
	if err != nil {
		return cspiCapacity, errors.Errorf(
			"failed to get pool %v properties for pool %s cmd out: %v error: %v",
			propertyList,
			poolName,
			valueList,
			err,
		)
	}
	// Since it was quarried in free, allocated and size output also
	// will be in same order.
	// valueList[0] contains value of free capacity in cStor pool
	// valueList[1] contains value of allocated capacity in cStor pool
	// valueList[2] contains total capacity of cStor pool
	freeSizeInBinarySI := GetCapacityInBinarySi(valueList[0])
	allocatedSizeInBinarySI := GetCapacityInBinarySi(valueList[1])
	totalSizeInBinarySI := GetCapacityInBinarySi(valueList[2])

	cspiCapacity.Free, err = GetCapacityFromString(freeSizeInBinarySI)
	if err != nil {
		return cspiCapacity, errors.Wrapf(err,
			"failed to parse pool free size %s of pool %s",
			freeSizeInBinarySI,
			poolName,
		)
	}
	cspiCapacity.Used, err = GetCapacityFromString(allocatedSizeInBinarySI)
	if err != nil {
		return cspiCapacity, errors.Wrapf(err,
			"failed to parse pool used size %s of pool %s",
			allocatedSizeInBinarySI,
			poolName,
		)
	}
	cspiCapacity.Total, err = GetCapacityFromString(totalSizeInBinarySI)
	if err != nil {
		return cspiCapacity, errors.Wrapf(err,
			"failed to parse pool total size %s of pool %s",
			totalSizeInBinarySI,
			poolName,
		)
	}
	return cspiCapacity, nil
}

// GetCapacityFromString will return value of given capacity in resource.Quantity form.
func GetCapacityFromString(capacity string) (resource.Quantity, error) {
	cap, err := resource.ParseQuantity(capacity)
	return cap, err
}

// GetCapacityInBinarySi replaces the unit to binary SI.
// zfs reports capacity in binary si i.e 1024 is the conversion factor.
// but the unit is K,M,G etc instead of Ki, Mi, Gi
// ToDO: This function currently only converts "K" --> "k" ( Ideally it should be "K" --> "Ki" and similarly
// ToDo: for other units. Revisit this.
func GetCapacityInBinarySi(capacity string) string {
	if strings.Contains(capacity, "K") {
		return strings.Replace(capacity, "K", "k", strings.Index(capacity, "K"))
	}
	return capacity
}

// SetPoolRDMode set the pool ReadOnly property based on the arrgument
func (oc *OperationsConfig) SetPoolRDMode(poolName string, isROMode bool) error {
	mode := "off"
	if isROMode {
		mode = "on"
	}
	ret, err := zfs.NewPoolSetProperty().
		WithProperty("io.openebs:readonly", mode).
		WithPool(poolName).
		WithExecutor(oc.zcmdExecutor).
		Execute()
	if err != nil {
		return errors.Errorf(
			"Failed to update readOnly mode to %s out:%v err:%v",
			mode, string(ret), err)
	}
	return nil

}
