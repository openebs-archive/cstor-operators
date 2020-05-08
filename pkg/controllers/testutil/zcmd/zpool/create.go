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
	"os"
	"strings"

	"github.com/pkg/errors"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	internalapi "github.com/openebs/api/pkg/internalapis/apis/cstor"
	"github.com/openebs/cstor-operators/pkg/controllers/common"
)

// Create mocks the zpool create command and will fill the topology
// according to the command triggered
func (mPool *MockPoolInfo) Create(cmd string) ([]byte, error) {
	if mPool.TestConfig.ZpoolCommand.ZpoolCreateError {
		return createError(cmd)
	}
	topology := mPool.buildTopologyFromCommand(cmd)
	mPool.Topology = topology
	mPool.PoolName = "cstor-" + os.Getenv(string(common.OpenEBSIOPoolName))
	values := strings.Split(cmd, "compression")
	if len(values) == 2 {
		partCommand := strings.TrimSpace(values[1])
		mPool.Compression = strings.Split(partCommand, " ")[0]
	}
	return []byte{}, nil
}

func createError(cmd string) ([]byte, error) {
	return []byte("fake error: active pool exists on the disks"), errors.Errorf("exit code 1")
}

// buildTopologyFromCommand returns the fake Vdev topology from command
func (mPoolInfo *MockPoolInfo) buildTopologyFromCommand(cmd string) *internalapi.Topology {
	var poolType string
	var writeCache bool
	var diskCount int
	raidGroupCount := -1
	topology := &internalapi.Topology{
		VdevTree: internalapi.VdevTree{
			VdevType: "root",
			//Some raw values
			VdevStats: vdevStats,
		},
	}
	values := strings.Split(cmd, " ")
	for i, s := range values {
		if poolType == "" && strings.ContainsAny(s, "/") {
			poolType = string(cstor.PoolStriped)
			mPoolInfo.DataRaidGroupType = poolType
		}
		if _, ok := supportedPoolTypes[s]; ok {
			poolType = s
			raidGroupCount++
			groupName := fmt.Sprintf("%s-%d", s, raidGroupCount)
			// Reset writeCache if there is another raidgroup
			if writeCache {
				if values[i-1] != "log" {
					writeCache = false
				} else {
					mPoolInfo.WriteCacheRaidGroupType = poolType
				}
			}
			topology.VdevTree.Topvdev = append(topology.VdevTree.Topvdev, getTopVdevFromRaidType(groupName, writeCache))
		}
		if s == "log" {
			writeCache = true
			// When user has writecache raid group then in command log will be followd by type
			poolType = ""
		}
		if strings.ContainsAny(s, "/") {
			if poolType == "stripe" {
				topology.VdevTree.Topvdev = append(topology.VdevTree.Topvdev, getVdevFromDisk(s, writeCache))
			} else {
				lenTopLevelVdev := len(topology.VdevTree.Topvdev) - 1
				topology.VdevTree.Topvdev[lenTopLevelVdev].Children = append(
					topology.VdevTree.Topvdev[lenTopLevelVdev].Children,
					getVdevFromDisk(s, writeCache),
				)
			}
			diskCount++
		}
	}
	mPoolInfo.DiskCount = diskCount
	topology.ChildrenCount = len(topology.VdevTree.Topvdev)

	return topology
}
