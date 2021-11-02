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
	"encoding/json"
	"fmt"

	internalapi "github.com/openebs/api/v3/pkg/internalapis/apis/cstor"
	"github.com/pkg/errors"
)

// Dump mocks zpool dump command and return output based on
// test configuration
func (poolMocker *PoolMocker) Dump(cmd string) ([]byte, error) {
	// If configuration expects error then return error
	if poolMocker.TestConfig.ZpoolCommand.ZpoolDumpError {
		return dumpError(cmd)
	}

	if poolMocker.PoolName == "" {
		return []byte{}, nil
	}
	if poolMocker.IsReplacementInProgress && poolMocker.TestConfig.ResilveringProgress == 0 {
		poolMocker.updateResilveringFinished(poolMocker.Topology.VdevTree.Topvdev)
		poolMocker.IsReplacementInProgress = false
	}
	encode, err := json.Marshal(poolMocker.Topology)
	if err != nil {
		return []byte(fmt.Sprintf("failed to parse data %s", err.Error())), errors.Errorf("exit status 1")
	}
	return encode, nil
}

func dumpError(cmd string) ([]byte, error) {
	return []byte("fake error"), errors.Errorf("exit status 1")
}

// updateResilveringFinished marks the resilvering process is completed if there is any
// resilvering marks present
func (poolMocker *PoolMocker) updateResilveringFinished(vdev []internalapi.Vdev) {
	for i, v := range vdev {
		if len(v.ScanStats) != 0 {
			// Marking as resilvering is finished
			vdev[i].VdevStats[internalapi.VdevScanProcessedIndex] = 0
			vdev[i].ScanStats = []uint64{}
		}
		for j, p := range v.Children {
			if len(p.ScanStats) != 0 {
				// Marking as resilvering is finished
				vdev[i].Children[j].VdevStats[internalapi.VdevScanProcessedIndex] = 0
				vdev[i].Children[j].ScanStats = []uint64{}
			}
			poolMocker.updateResilveringFinished(p.Children)
		}
	}
}
