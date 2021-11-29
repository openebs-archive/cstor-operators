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

import internalapi "github.com/openebs/api/v3/pkg/internalapis/apis/cstor"

var (
	supportedPoolTypes = map[string]bool{
		"stripe": true,
		"mirror": true,
		"raidz":  true,
		"raidz2": true,
	}
	vdevStats = []uint64{29352945445, 7, 0, 264704, 10670309376, 10670309376, 0, 0, 0, 3, 160, 0, 0, 0, 0, 24576, 2181632, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

// getTopVdevFeomRaidType returns the new instance of top vdev
func getTopVdevFromRaidType(raidType string, isWriteCache bool) internalapi.Vdev {
	writeCache := 0
	if isWriteCache {
		writeCache = 1
	}
	vdev := internalapi.Vdev{
		VdevType:  raidType,
		VdevStats: vdevStats,
		IsLog:     writeCache,
	}
	return vdev
}

// getVdevFromDisk returns the new instance of child vdev
func getVdevFromDisk(diskPath string, isWriteCache bool) internalapi.Vdev {
	writeCache := 0
	childStats := []uint64{4839818939, 7, 0, 0, 0, 0, 10737418240, 0, 0, 3, 260, 0, 0, 0, 0, 24576, 2822656, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if isWriteCache {
		writeCache = 1
	}
	vdev := internalapi.Vdev{
		VdevType:  "file",
		VdevStats: childStats,
		Path:      diskPath,
		IsLog:     writeCache,
	}
	return vdev
}
