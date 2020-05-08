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
	"os"

	"github.com/openebs/cstor-operators/pkg/controllers/common"
	"github.com/pkg/errors"
)

const (
	// PoolPrefix is prefix for pool name
	PoolPrefix string = "cstor-"
)

var poolName string

// ErrorWrapf wrap error
// If given err is nil then it will return new error
func ErrorWrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return errors.Errorf(format, args...)
	}

	return errors.Wrapf(err, format, args...)
}

// PoolName return pool name for given CSPI object
func PoolName() string {
	if poolName == "" {
		poolName = PoolPrefix + os.Getenv(string(common.OpenEBSIOPoolName))
	}
	return poolName
}

// IsEmpty check if string is empty or not
func IsEmpty(s string) bool {
	return len(s) == 0
}
