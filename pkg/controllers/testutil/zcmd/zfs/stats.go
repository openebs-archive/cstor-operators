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
	"encoding/json"
	"fmt"

	zstats "github.com/openebs/cstor-operators/pkg/zcmd/zfs/stats"
	"github.com/pkg/errors"
)

var (
	// Below are possible states reported by zfs
	replicaStates = []string{"Rebuilding", "Degraded", "Offline", ""}
)

// GetStats mocks the zfs stats command and returns the error based on the output
func (volumeMocker *VolumeMocker) GetStats(cmd string) ([]byte, error) {
	if volumeMocker.TestConfig.ZFSCommand.ZFSStatsError {
		return []byte("fake error to get values"), errors.Errorf("exit status 1")
	}
	zStats := zstats.ZFSStats{}
	for i := 0; i < volumeMocker.TestConfig.ProvisionedReplicas; i++ {
		index := i % len(replicaStates)
		name := fmt.Sprintf("ProvisionedVolume-%d", i)
		zStats.Stats = append(zStats.Stats, zstats.Stats{Name: name, Status: replicaStates[index]})
	}
	for i := 0; i < volumeMocker.TestConfig.HealthyReplicas; i++ {
		name := fmt.Sprintf("HealthyVolume-%d", i)
		zStats.Stats = append(zStats.Stats, zstats.Stats{Name: name, Status: "Healthy"})
	}
	return json.Marshal(zStats)
}
