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
package zfs

import (
	"fmt"

	"github.com/pkg/errors"
)

// ListProperty mocks the zfs list command
func (volumeMocker *VolumeMocker) ListProperty(cmd string) ([]byte, error) {
	if volumeMocker.TestConfig.ZFSCommand.ZFSListError {
		return []byte("fake zfs error"), errors.New("exit statu 1")
	}
	var output []string
	for i := 0; i < volumeMocker.TestConfig.ProvisionedReplicas; i++ {
		output = append(output, fmt.Sprintf("%s/ProvisionedVolume-%d\n", volumeMocker.PoolName, i))
	}
	for i := 0; i < volumeMocker.TestConfig.HealthyReplicas; i++ {
		output = append(output, fmt.Sprintf("%s/HealthyVolume-%d\n", volumeMocker.PoolName, i))
	}
	return []byte(fmt.Sprintf("%s", output)), nil
}
