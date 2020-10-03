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

package cstorvolumeconfig

// backupHelper is an interface which will serve
// the request of different versions of backup resources
type backupHelper interface {
	isBackupCompleted() bool
	getCSPIName() string
	findLastBackupStat() string
	updateBackupStatus(string) backupHelper
	getBackupObject() interface{}
	//TODO: Rename the function
	deleteCompletedBackup(name, namespace, snapName string) error
	deleteBackup(name, namespace string) error
	getOrCreateLastBackupSnap() (string, error)
	setBackupStatus(string) backupHelper
	setLastSnapshotName(string) backupHelper
	createBackupResource() (backupHelper, error)
}
