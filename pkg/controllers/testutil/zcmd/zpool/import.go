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
	"strings"

	"github.com/pkg/errors"
)

// Import mocks zpool import command
func (poolMocker *PoolMocker) Import(cmd string) ([]byte, error) {
	// If configuration expects error then return error
	if poolMocker.TestConfig.ZpoolCommand.ZpoolImportError {
		return importError(cmd)
	}

	if len(strings.Split(cmd, " ")) == 2 {
		return []byte{}, nil
	}
	if poolMocker.PoolName == "" {
		return []byte("no pools available to import"), errors.Errorf("exit status 1")
	}
	poolMocker.IsPoolImported = true
	return []byte{}, nil
}

func importError(cmd string) ([]byte, error) {
	return []byte("Fake No Pool Exist to import"), nil
}
