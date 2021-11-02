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
	"math/rand"
	"strings"
	"time"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/pkg/errors"
)

// Add mocks the zpool add command and returns error based on the test configuration
func (poolMocker *PoolMocker) Add(cmd string) ([]byte, error) {
	if poolMocker.PoolName == "" {
		return []byte("cannot open 'pool': no such pool"), errors.Errorf("exit status 1")
	}
	// If configuration expects error then return error
	if poolMocker.TestConfig.ZpoolCommand.ZpoolAddError {
		return addError(cmd)
	}
	poolMocker.addVdev(cmd)
	return []byte{}, nil
}

// addVdev adds the new vdev/devices into the pool topology
func (poolMocker *PoolMocker) addVdev(cmd string) {
	var poolType string
	var isWriteCache bool
	values := strings.Split(cmd, " ")
	for i, s := range values {
		if poolType == "" && strings.ContainsAny(s, "/") {
			poolType = string(cstor.PoolStriped)
		}
		if _, ok := supportedPoolTypes[s]; ok {
			if isWriteCache {
				if values[i-1] != "log" {
					isWriteCache = false
				}
			}
			if s != "stripe" {
				rand.Seed(time.Now().UnixNano())
				groupName := fmt.Sprintf("%s-%d", s, rand.Intn(21))
				poolMocker.Topology.VdevTree.Topvdev = append(
					poolMocker.Topology.VdevTree.Topvdev,
					getTopVdevFromRaidType(groupName, isWriteCache))
			}
		}
		if s == "log" {
			isWriteCache = true
			poolType = ""
		}
		if strings.ContainsAny(s, "/") {
			if poolType == "stripe" {
				poolMocker.Topology.VdevTree.Topvdev = append(
					poolMocker.Topology.VdevTree.Topvdev,
					getVdevFromDisk(s, isWriteCache))
			} else {
				lenTopLevelVdev := len(poolMocker.Topology.VdevTree.Topvdev) - 1
				poolMocker.Topology.VdevTree.Topvdev[lenTopLevelVdev].Children = append(
					poolMocker.Topology.VdevTree.Topvdev[lenTopLevelVdev].Children,
					getVdevFromDisk(s, isWriteCache),
				)
			}
			poolMocker.DiskCount++
		}
	}
}

// addError returns fake error if test configuration is expecting error
func addError(cmd string) ([]byte, error) {
	return []byte("fake error"), errors.Errorf("exit status 1")
}
