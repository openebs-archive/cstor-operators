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
	internalapi "github.com/openebs/api/v3/pkg/internalapis/apis/cstor"
)

// PoolMocker mocks the zpool utitlity commands
type PoolMocker struct {
	// Topology holds the vdev topology of the pool
	Topology *internalapi.Topology
	// PoolName holds the cstor pool name
	PoolName string
	// DataRaidGroupType represents the type of the data raid group
	DataRaidGroupType string
	// WriteCacheRaidGroupType represents the type of the write cache raid group
	WriteCacheRaidGroupType string
	// Compression holds the type of pool compression
	Compression string
	// IsPoolImported used to know whether pool is imported or not
	IsPoolImported bool
	// IsPoolReadOnlyMode informs pool ReadOnly mode
	IsPoolReadOnlyMode bool
	// IsReplacementInProgress the status of replacement operation
	IsReplacementInProgress bool
	// DiskCount represents the total no.of disks present in the pool
	DiskCount int
	// TestConfig holds the test related information
	TestConfig TestConfig
}

// TestConfig holds the the test configuration based on this
// configuration zpool utility commands will return error
type TestConfig struct {
	ZpoolCommand ZpoolCommandError
	// ResilveringProgress represents fake resilvering progress
	// If the value is 0 then zpool dump marks vdev as resilvering
	// completed
	ResilveringProgress int
}

// ZpoolCommandError used to inject the errors in various Zpool commands
// It will help to mock the zpool command behaviour
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
