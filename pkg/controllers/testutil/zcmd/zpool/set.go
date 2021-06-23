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
package zpool

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// SetProperty mocks the zpool get command and returns the error based on the output
func (poolMocker *PoolMocker) SetProperty(cmd string) ([]byte, error) {

	// If configuration expects error then return error
	if poolMocker.TestConfig.ZpoolCommand.ZpoolGetError {
		return getPropertyError(cmd)
	}

	values := strings.Split(cmd, " ")
	if poolMocker.PoolName == "" {
		return []byte(fmt.Sprintf("cannot open '%s': no such pool", values[len(values)-1])), errors.Errorf("exit statu 1")
	}
	if !strings.Contains(cmd, poolMocker.PoolName) {
		return []byte(fmt.Sprintf("cannot open '%s': no such pool", values[len(values)-1])), errors.Errorf("exit statu 1")
	}
	// Add fields in PoolMocker for setting the property
	return []byte{}, nil
}
