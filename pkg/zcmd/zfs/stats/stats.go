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
