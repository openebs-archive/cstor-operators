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

// VolumeMocker contains the volume information which
// will helpful to execute zfs command
type VolumeMocker struct {
	PoolName    string
	Compression string
	// TestConfig holds the Volume test related information
	TestConfig TestConfig
}

// TestConfig holds the the test configuration based on this
// configuration zfs utility commands will return error
type TestConfig struct {
	ZFSCommand          ZFSCommandError
	HealthyReplicas     int
	ProvisionedReplicas int
}

// ZfsCommandError used to inject the errors in various ZFS commands
// It will help to mock the zfs command behaviour
type ZFSCommandError struct {
	ZFSStatsError bool
	ZFSGetError   bool
	ZFSListError  bool
}
