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

package v1

// IsFailed returns true if backup is failed
func (backup *CStorBackup) IsFailed() bool {
	return backup.Status == BKPCStorStatusFailed
}

// IsSucceeded returns true if backup completed successfully
func (backup *CStorBackup) IsSucceeded() bool {
	return backup.Status == BKPCStorStatusDone
}

// IsPending returns true if the backup is in pending state
func (backup *CStorBackup) IsPending() bool {
	return backup.Status == BKPCStorStatusPending
}

// IsInProgress returns true if the backup is in progress state
func (backup *CStorBackup) IsInProgress() bool {
	return backup.Status == BKPCStorStatusInProgress
}

// IsInInit returns true if the backup is in init state
func (backup *CStorBackup) IsInInit() bool {
	return backup.Status == BKPCStorStatusInit
}

// --------------------- Restore ------------------- //

// IsSucceeded return true if restore is done
func (restore *CStorRestore) IsSucceeded() bool {
	return restore.Status == RSTCStorStatusDone
}

// IsFailed return true if restore is failed
func (restore *CStorRestore) IsFailed() bool {
	return restore.Status == RSTCStorStatusFailed
}

// IsInProgress return true if restore is in progress
func (restore *CStorRestore) IsInProgress() bool {
	return restore.Status == RSTCStorStatusInProgress
}

// IsPending return true if restore is in pending
func (restore *CStorRestore) IsPending() bool {
	return restore.Status == RSTCStorStatusPending
}

// IsInInit return true if restore is in init status
func (restore *CStorRestore) IsInInit() bool {
	return restore.Status == RSTCStorStatusInit
}
