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

package cstor

import (
	"encoding/json"
	"strings"

	"github.com/openebs/api/v3/pkg/util"
)

/*
* enum maintaing in cstor data plane side
* typedef enum dsl_scan_state {
*        DSS_NONE,
*        DSS_SCANNING,
*        DSS_FINISHED,
*        DSS_CANCELED,
*        DSS_NUM_STATES
*} dsl_scan_state_t;
 */

//TODO: Improve comments during review process

//PoolScanState states various pool scan states
type PoolScanState uint64

const (
	// PoolScanNone represents pool scanning is not yet started
	PoolScanNone PoolScanState = iota
	// PoolScanScanning represents pool is undergoing scanning
	PoolScanScanning
	// PoolScanFinished represents pool scanning is finished
	PoolScanFinished
	// PoolScanCanceled represents pool scan is aborted
	PoolScanCanceled
	// PoolScanNumOfStates holds value 4
	PoolScanNumOfStates
)

// PoolScanFunc holds various scanning functions
type PoolScanFunc uint64

const (
	// PoolScanFuncNone holds value 0
	PoolScanFuncNone PoolScanFunc = iota
	// PoolScanFuncScrub holds value 1
	PoolScanFuncScrub
	// PoolScanFuncResilver holds value 2 which states device under went resilvering
	PoolScanFuncResilver
	// PoolScanFuncStates holds value 3
	PoolScanFuncStates
)

// NOTE: 1. VdevState represent the state of the vdev/disk in pool.
//       2. VdevAux represents gives the reason why disk/vdev are in that state.

// VdevState represent various device/disk states
type VdevState uint64

/*
 * vdev states are ordered from least to most healthy.
 * Link: https://github.com/openebs/cstor/blob/f0896898a0be2102e2865cf44b16b88c91f6bb91/include/sys/fs/zfs.h#L723
 */
const (
	// VdevStateUnknown represents uninitialized vdev
	VdevStateUnknown VdevState = iota
	// VdevStateClosed represents vdev currently not opened
	VdevStateClosed
	// VdevStateOffline represents vdev not allowed to open
	VdevStateOffline
	// VdevStateRemoved represents vdev explicitly removed from system
	VdevStateRemoved
	// VdevStateCantOpen represents tried to open, but failed
	VdevStateCantOpen
	// VdevStateFaulted represents external request to fault device
	VdevStateFaulted
	// VdevStateDegraded represents Replicated vdev with unhealthy kids
	VdevStateDegraded
	// VdevStateHealthy represents vdev is presumed good
	VdevStateHealthy
)

// VdevAux represents reasons why vdev can't open
type VdevAux uint64

/*
 * vdev aux states.  When a vdev is in the CANT_OPEN state, the aux field
 * of the vdev stats structure uses these constants to distinguish why.
 */
// NOTE: Added only required enums for more information please have look at
// https://github.com/openebs/cstor/blob/f0896898a0be2102e2865cf44b16b88c91f6bb91/include/sys/fs/zfs.h#L740
const (
	VdevAuxNone            VdevAux = iota /* no error */
	VdevAuxOpenFailed                     /* ldi_open_*() or vn_open() failed  */
	VdevAuxCorruptData                    /* bad label or disk contents           */
	VdevAuxNoReplicas                     /* insufficient number of replicas      */
	VdevAuxBadGUIDSum                     /* vdev guid sum doesn't match          */
	VdevAuxTooSmall                       /* vdev size is too small               */
	VdevAuxBadLabel                       /* the label is OK but invalid          */
	VdevAuxVersionNewer                   /* on-disk version is too new           */
	VdevAuxVersionOlder                   /* on-disk version is too old           */
	VdevAuxUnSupFeat                      /* unsupported features                 */
	VdevAuxSpared                         /* hot spare used in another pool       */
	VdevAuxErrExceeded                    /* too many errors                      */
	VdevAuxIOFailure                      /* experienced I/O failure              */
	VdevAuxBadLog                         /* cannot read log chain(s)             */
	VdevAuxExternal                       /* external diagnosis or forced fault   */
	VdevAuxSplitPool                      /* vdev was split off into another pool */
	VdevAuxBadAShift                      /* vdev ashift is invalid               */
	VdevAuxExternalPersist                /* persistent forced fault      */
	VdevAuxActive                         /* vdev active on a different host      */
)

const (
	// PoolOperator is the name of the tool that makes pool-related operations.
	PoolOperator = "zpool"
	// VdevScanProcessedIndex is index of scaned bytes on disk
	VdevScanProcessedIndex = 25
	// VdevScanStatsStateIndex represents the index of dataset scan state
	VdevScanStatsStateIndex = 1
	// VdevScanStatsScanFuncIndex point to index which inform whether device
	// under went resilvering or not
	VdevScanStatsScanFuncIndex = 0
	// VdevStateIndex represents the device state information
	VdevStateIndex = 1
	// VdevAuxIndex represents vdev aux states. When a vdev is
	// in the CANT_OPEN state, the aux field of the vdev stats
	// structure uses these constants to distinguish why
	VdevAuxIndex = 2
)

// Topology contains the topology strucure of disks used in backend
type Topology struct {
	// Number of top-level children in topology (doesnt include spare/l2cache)
	ChildrenCount int `json:"vdev_children,omitempty"`

	// Root of vdev topology
	VdevTree VdevTree `json:"vdev_tree,omitempty"`
}

// VdevTree contains the tree strucure of disks used in backend
type VdevTree struct {
	// root for Root vdev, Raid type in case of non-level 0 vdev,
	// and file/disk in case of level-0 vdev
	VdevType string `json:"type,omitempty"`

	// top-level vdev topology
	Topvdev []Vdev `json:"children,omitempty"`

	// list of read-cache devices
	Readcache []Vdev `json:"l2cache,omitempty"`

	// list of spare devices
	Spares []Vdev `json:"spares,omitempty"`

	// vdev indetailed statistics
	VdevStats []uint64 `json:"vdev_stats,omitempty"`

	// ScanStats states replaced device scan state
	ScanStats []uint64 `json:"scan_stats,omitempty"`
}

// Vdev relates to a logical or physical disk in backend
type Vdev struct {
	// root for Root vdev, Raid type in case of non-level 0 vdev,
	// and file/disk in case of level-0 vdev
	VdevType string `json:"type,omitempty"`

	// Path of the disk or sparse file
	Path string `json:"path,omitempty"`

	// 0 means not write-cache device, 1 means write-cache device
	IsLog int `json:"is_log,omitempty"`

	// 0 means not spare device, 1 means spare device
	IsSpare int `json:"is_spare,omitempty"`

	// 0 means partitioned disk, 1 means whole disk
	IsWholeDisk int `json:"whole_disk,omitempty"`

	// Capacity represents the size of the disk used in pool
	Capacity uint64 `json:"asize,omitempty"`

	// vdev indetailed statistics
	VdevStats []uint64 `json:"vdev_stats,omitempty"`

	ScanStats []uint64 `json:"scan_stats,omitempty"`

	// child vdevs of the logical disk or null for physical disk/sparse
	Children []Vdev `json:"children,omitempty"`
}

// VdevList is alias of list of Vdevs
type VdevList []Vdev

// Dump runs 'zpool dump' command and unmarshal the output in above schema
func Dump() (Topology, error) {
	var t Topology
	runnerVar := util.RealRunner{}
	out, err := runnerVar.RunCombinedOutput(PoolOperator, "dump")
	if err != nil {
		return t, err
	}
	err = json.Unmarshal(out, &t)
	return t, err
}

// GetVdevFromPath returns vdev if provided path exists in vdev topology
func (l VdevList) GetVdevFromPath(path string) (Vdev, bool) {
	for _, v := range l {
		if strings.EqualFold(path, v.Path) {
			return v, true
		}
		for _, p := range v.Children {
			if strings.EqualFold(path, p.Path) {
				return p, true
			}
			if vdev, r := VdevList(p.Children).GetVdevFromPath(path); r {
				return vdev, true
			}
		}
	}
	return Vdev{}, false
}

// GetVdevState returns current state of Vdev
// NOTE: Below function is taken from openebs/cstor
// https://github.com/openebs/cstor/blob/f0896898a0be2102e2865cf44b16b88c91f6bb91/lib/libzfs/libzfs_pool.c#L183<Paste>
func (v Vdev) GetVdevState() string {
	state := v.VdevStats[VdevStateIndex]
	aux := v.VdevStats[VdevAuxIndex]
	switch state {
	case uint64(VdevStateClosed):
		fallthrough
	case uint64(VdevStateOffline):
		return "OFFLINE"
	case uint64(VdevStateRemoved):
		return "REMOVED"
	case uint64(VdevStateCantOpen):
		if aux == uint64(VdevAuxCorruptData) || aux == uint64(VdevAuxBadLog) {
			return "FAULTED"
		} else if aux == uint64(VdevAuxSplitPool) {
			return "SPLIT"
		} else {
			return "UNAVAILABLE"
		}
	case uint64(VdevStateFaulted):
		return "FAULTED"
	case uint64(VdevStateDegraded):
		return "DEGRADED"
	case uint64(VdevStateHealthy):
		return "ONLINE"
	}
	return "UNKNOWN"
}
