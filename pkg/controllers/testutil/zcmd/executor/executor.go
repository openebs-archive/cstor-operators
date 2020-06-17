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

package executor

import (
	"fmt"
	"strings"

	zfs "github.com/openebs/cstor-operators/pkg/controllers/testutil/zcmd/zfs"
	zpool "github.com/openebs/cstor-operators/pkg/controllers/testutil/zcmd/zpool"
	"github.com/pkg/errors"
)

// FakeZcmd holds the Pool and Volumes information
// which will helpful to mocking zpool and zfs commands
type FakeZcmd struct {
	poolMocker   *zpool.PoolMocker
	volumeMocker *zfs.VolumeMocker
}

// NewFakeZCommand returns new instance of FakeZcmd
func NewFakeZCommand() *FakeZcmd {
	return &FakeZcmd{
		poolMocker:   &zpool.PoolMocker{},
		volumeMocker: &zfs.VolumeMocker{},
	}
}

// NewFakeZCommandFromMockers returns new instance of FakeZcmd from MockPoolInfo
func NewFakeZCommandFromMockers(poolMocker *zpool.PoolMocker,
	volumeMocker *zfs.VolumeMocker) *FakeZcmd {
	return &FakeZcmd{
		poolMocker:   poolMocker,
		volumeMocker: volumeMocker,
	}
}

// Execute is to execute fake ZPOOL/ZFS commands which may
// return output (or) error based on test configuration
func (f *FakeZcmd) Execute(cmd string) ([]byte, error) {
	if strings.Contains(cmd, "zpool") {
		return f.executePoolCommands(cmd)
	} else if strings.Contains(cmd, "zfs") {
		return f.executeVolumeCommands(cmd)
	}
	return []byte(fmt.Sprintf("please mock %s command", cmd)), errors.Errorf("exit status 1")
}

func (f *FakeZcmd) executePoolCommands(cmd string) ([]byte, error) {
	cmd = strings.TrimSpace(cmd)
	values := strings.Split(cmd, " ")
	switch values[1] {
	case "create":
		return f.poolMocker.Create(cmd)
	case "import":
		return f.poolMocker.Import(cmd)
	case "get":
		return f.poolMocker.GetProperty(cmd)
	case "destroy":
		return f.poolMocker.Delete(cmd)
	case "dump":
		return f.poolMocker.Dump(cmd)
	case "add":
		return f.poolMocker.Add(cmd)
	case "labelclear":
		return f.poolMocker.LabelClear(cmd)
	case "replace":
		return f.poolMocker.Replace(cmd)
	case "set":
		return f.poolMocker.SetProperty(cmd)
	}
	return []byte(fmt.Sprintf("Please mock zpool %s command", values[1])), errors.Errorf("exit status 1")
}

func (f *FakeZcmd) executeVolumeCommands(cmd string) ([]byte, error) {
	f.volumeMocker.PoolName = f.poolMocker.PoolName
	cmd = strings.TrimSpace(cmd)
	values := strings.Split(cmd, " ")
	switch values[1] {
	case "get":
		return f.volumeMocker.GetProperty(cmd)
	case "list":
		return f.volumeMocker.ListProperty(cmd)
	case "stats":
		return f.volumeMocker.GetStats(cmd)
	}
	return []byte(fmt.Sprintf("Please mock zfs %s command", values[1])), errors.Errorf("exit status 1")
}
