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

	internalapi "github.com/openebs/api/v3/pkg/internalapis/apis/cstor"
	"github.com/pkg/errors"
)

var (
	resilveringVdevStats = []uint64{uint64(internalapi.PoolScanFuncResilver), uint64(internalapi.PoolScanFinished), 0, 0, 0, 1234, 1203}
)

// Replace mocks the zpool replace command and retutns error based on the test
// configuration
func (poolMocker *PoolMocker) Replace(cmd string) ([]byte, error) {
	// If configuration expects error then return error
	if poolMocker.TestConfig.ZpoolCommand.ZpoolReplaceError {
		return replaceError(cmd)
	}
	// zpool replace <pool_name> <old_path> <new_path>
	values := strings.Split(cmd, "replace")
	if len(values) == 2 {
		paths := strings.Split(strings.TrimSpace(values[1]), " ")
		if len(paths) < 3 {
			return []byte("inappropriate command"), errors.Errorf("exit status 1")
		}
		// paths contains: paths[0] -- PoolName; paths[1] -- oldDevlink; paths[3] -- newDevLink
		err := poolMocker.replacePathInVdev(paths[1], paths[2], poolMocker.Topology.VdevTree.Topvdev)
		if err != nil {
			return []byte(err.Error()), errors.Errorf("exit status 1")
		}
	}
	poolMocker.IsReplacementInProgress = true
	return []byte{}, nil
}

// replacePathInVdev replace the old path with new path in Topology
func (poolMocker *PoolMocker) replacePathInVdev(oldPath, newPath string, vdev []internalapi.Vdev) error {
	for i, v := range vdev {
		if v.Path == oldPath {
			vdev[i].Path = newPath
			// Marking as resilvering is in progress
			vdev[i].VdevStats[internalapi.VdevScanProcessedIndex] = 1223
			vdev[i].ScanStats = resilveringVdevStats
			return nil
		}
		for j, p := range v.Children {
			if p.Path == oldPath {
				vdev[i].Children[j].Path = newPath
				// Marking as resilvering is in progress
				vdev[i].Children[j].VdevStats[internalapi.VdevScanProcessedIndex] = 1223
				vdev[i].Children[j].ScanStats = resilveringVdevStats
				return nil
			}
			if err := poolMocker.replacePathInVdev(oldPath, newPath, p.Children); err == nil {
				return nil
			}
		}
	}
	return errors.Errorf("oldpath doesn't exist in pool")
}

func replaceError(cmd string) ([]byte, error) {
	return []byte("fake error can't replace vdev"), errors.Errorf("exit status 1")
}
