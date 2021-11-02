/*
Copyright 2018 The OpenEBS Authors.

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

package cspicontroller

import (
	"os"
	"reflect"

	types "github.com/openebs/api/v3/pkg/apis/types"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/pkg/errors"
)

const (
	// PoolPrefix is prefix for pool name
	PoolPrefix string = "cstor-"
	// OpenEBSIOCSPIID is the environment variable specified in pod.
	// It holds the UID of the CSPI
	OpenEBSIOCSPIID string = "OPENEBS_IO_CSPI_ID"
)

// IsRightCStorPoolInstanceMgmt is to check if the pool request is for this pod.
func IsRightCStorPoolInstanceMgmt(cspi *cstor.CStorPoolInstance) bool {
	return os.Getenv(OpenEBSIOCSPIID) == string(cspi.GetUID())
}

// IsStatusChange is to check only status change of cStorPoolInstance object.
func IsStatusChange(oldStatus, newStatus cstor.CStorPoolInstanceStatus) bool {
	return !reflect.DeepEqual(oldStatus, newStatus)
}

// IsReconcileDisabled check if reconciliation is disabled for given object or not
func IsReconcileDisabled(cspi *cstor.CStorPoolInstance) bool {
	return cspi.HasAnnotation(types.OpenEBSDisableReconcileLabelKey, "true")
}

// IsEmpty check if string is empty or not
func IsEmpty(s string) bool {
	return len(s) == 0
}

// ErrorWrapf wrap error
// If given err is nil then it will return new error
func ErrorWrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return errors.Errorf(format, args...)
	}

	return errors.Wrapf(err, format, args...)
}
