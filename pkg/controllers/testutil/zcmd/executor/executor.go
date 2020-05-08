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

// FakeZcmd contains the information about Pool and Volumes
// which will helpful to mocking zpool and zfs commands
type FakeZcmd struct {
	mPoolInfo   *zpool.MockPoolInfo
	mVolumeInfo *zfs.MockVolumeInfo
}

// NewFakeZCommand returns new instance of FakeZcmd
func NewFakeZCommand() *FakeZcmd {
	return &FakeZcmd{
		mPoolInfo:   &zpool.MockPoolInfo{},
		mVolumeInfo: &zfs.MockVolumeInfo{},
	}
}

// NewFakeZCommandFromPoolInfo returns new instance of FakeZcmd from MockPoolInfo
func NewFakeZCommandFromPoolInfo(mPoolInfo *zpool.MockPoolInfo) *FakeZcmd {
	return &FakeZcmd{
		mPoolInfo:   mPoolInfo,
		mVolumeInfo: &zfs.MockVolumeInfo{},
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
		return f.mPoolInfo.Create(cmd)
	case "import":
		return f.mPoolInfo.Import(cmd)
	case "get":
		return f.mPoolInfo.GetProperty(cmd)
	case "destroy":
		return f.mPoolInfo.Delete(cmd)
	case "dump":
		return f.mPoolInfo.Dump(cmd)
	case "add":
		return f.mPoolInfo.Add(cmd)
	case "labelclear":
		return f.mPoolInfo.LabelClear(cmd)
	case "replace":
		return f.mPoolInfo.Replace(cmd)
	case "set":
		return f.mPoolInfo.SetProperty(cmd)
	}
	return []byte(fmt.Sprintf("Please mock zpool %s command", values[1])), errors.Errorf("exit status 1")
}

func (f *FakeZcmd) executeVolumeCommands(cmd string) ([]byte, error) {
	cmd = strings.TrimSpace(cmd)
	values := strings.Split(cmd, " ")
	switch values[1] {
	case "get":
		return f.mVolumeInfo.GetProperty(cmd)
	}
	return []byte(fmt.Sprintf("Please mock zfs %s command", values[1])), errors.Errorf("exit status 1")
}
