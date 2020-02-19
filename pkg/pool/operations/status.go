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
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"strings"

	zfs "github.com/openebs/cstor-operators/pkg/zcmd"
)

// GetPropertyValue will return value of given property for given pool
func GetPropertyValue(poolName, property string) (string, error) {
	ret, err := zfs.NewPoolGetProperty().
		WithScriptedMode(true).
		WithField("value").
		WithProperty(property).
		WithPool(poolName).
		Execute()
	if err != nil {
		return "", err
	}
	outStr := strings.Split(string(ret), "\n")
	return outStr[0], nil
}

func GetPoolCapacity(poolName, capacityProperty string) (resource.Quantity, error) {
	size, err := GetPropertyValue(poolName, capacityProperty)
	if err != nil {
		return resource.Quantity{}, errors.Wrapf(err, "failed to get pool %s size for pool %s", capacityProperty,poolName)
	}
	sizeInBinarySI := GetCapacityInBinarySi(size)

	poolSize, err := GetCapacityFromString(sizeInBinarySI)
	if err != nil {
		return resource.Quantity{}, errors.Wrapf(err, "failed to get parse pool free size for pool %s", poolName)
	}
	return poolSize, nil
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
	if strings.Contains(capacity,"K"){
		return strings.Replace(capacity, "K", "k", strings.Index(capacity, "K"))
	}
	return capacity
}
