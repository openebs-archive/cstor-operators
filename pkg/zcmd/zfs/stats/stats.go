/*
Copyright 2020 The OpenEBS Authors
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
package vstats

// ZFSStats used to represents zfs dataset stats
type ZFSStats struct {
	// Stats is an array which holds zfs volume related stats
	Stats []Stats `json:"stats"`
}

// Stats contain the zfs volume dataset related stats
type Stats struct {
	// Name of the zfs volume. Usually naming convention
	// will be pool_name/volume_name
	Name string `json:"name"`
	// Status of the zfs volume like Healthy, Degraded, Offline, Error
	Status string `json:"status"`
	// RebuildStatus of the zfs volume dataset. Following are possible states
	/*
		// rebuilding can be initiated
		ZVOL_REBUILDING_INIT
		// zvol is rebuilding snapshots
		ZVOL_REBUILDING_SNAP
		// zvol is rebuilding active dataset
		ZVOL_REBUILDING_AFS
		// Rebuilding completed with success
		ZVOL_REBUILDING_DONE
		// errored during rebuilding, but not completed
		ZVOL_REBUILDING_ERRORED
		// Rebuilding completed with error
		ZVOL_REBUILDING_FAILED
	*/
	RebuildStatus             string `json:"rebuildStatus"`
	IsIOAckSenderCreated      int    `json:"isIOAckSenderCreated"`
	IsIOReceiverCreated       int    `json:"isIOReceiverCreated"`
	RunningIONum              int    `json:"runningIONum"`
	CheckpointedIONum         int    `json:"checkpointedIONum"`
	DegradedCheckpointedIONum int    `json:"degradedCheckpointedIONum"`
	CheckpointedTime          int    `json:"checkpointedTime"`
	RebuildBytes              int    `json:"rebuildBytes"`
	RebuildCnt                int    `json:"rebuildCnt"`
	RebuildDoneCnt            int    `json:"rebuildDoneCnt"`
	RebuildFailedCnt          int    `json:"rebuildFailedCnt"`
	Quorum                    int    `json:"quorum"`
}
