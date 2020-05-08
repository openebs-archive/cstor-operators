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
	internalapi "github.com/openebs/api/pkg/internalapis/apis/cstor"
)

// MockPoolInfo contains the pool information which
// will helpful to return error
type MockPoolInfo struct {
	Topology                *internalapi.Topology
	PoolName                string
	DataRaidGroupType       string
	WriteCacheRaidGroupType string
	Compression             string
	IsPoolImported          bool
	IsPoolReadOnlyMode      bool
	IsReplacementTriggered  bool
	DiskCount               int
	TestConfig              TestConfig
}

// TestConfig holds the the test configuration based on this
// configuration mocked zpool will return error
type TestConfig struct {
	ZpoolCommand ZpoolCommandError
	// ResilveringProgress represents fake resilvering progress
	// If the value is 0 then zpool dump marks vdev as resilvering
	// completed
	ResilveringProgress int
}

// ZpoolCommandError used to inject the errors in various Zpool commands
type ZpoolCommandError struct {
	ZpoolAddError        bool
	ZpoolClearError      bool
	ZpoolDestroyError    bool
	ZpoolDumpError       bool
	ZpoolGetError        bool
	ZpoolLabelClearError bool
	ZpoolOnlineError     bool
	ZpoolReplaceError    bool
	ZpoolStatusError     bool
	ZpoolAttachError     bool
	ZpoolCreateError     bool
	ZpoolDetachError     bool
	ZpoolExportError     bool
	ZpoolImportError     bool
	ZpoolOfflineError    bool
	ZpoolRemoveError     bool
	ZpoolSetError        bool
}
