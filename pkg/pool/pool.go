/*
Copyright 2018 The OpenEBS Authors.

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

package pool

import (
	"strings"
	"time"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	zpool "github.com/openebs/api/v3/pkg/internalapis/apis/cstor"
	"github.com/openebs/api/v3/pkg/util"
	zfs "github.com/openebs/cstor-operators/pkg/zcmd"
	"k8s.io/klog"
)

// PoolOperator is the name of the tool that makes pool-related operations.
const (
	StatusNoPoolsAvailable = "no pools available"
	ZpoolStatusDegraded    = "DEGRADED"
	ZpoolStatusFaulted     = "FAULTED"
	ZpoolStatusOffline     = "OFFLINE"
	ZpoolStatusOnline      = "ONLINE"
	ZpoolStatusRemoved     = "REMOVED"
	ZpoolStatusUnavail     = "UNAVAIL"
)

//PoolAddEventHandled is a flag representing if the pool has been initially imported or created
var PoolAddEventHandled = false

// PoolNamePrefix is a typed string to store pool name prefix
type PoolNamePrefix string

// ImportedCStorPools is a map of imported cstor pools API config identified via their UID
var ImportedCStorPools map[string]*cstor.CStorPoolInstance

// CStorZpools is a map of imported cstor pools config at backend identified via their UID
var CStorZpools map[string]zpool.Topology

// PoolPrefix is prefix for pool name
const (
	PoolPrefix PoolNamePrefix = "cstor-"
)

// ImportOptions contains the options to build import command
type ImportOptions struct {
	// CachefileFlag option to use cachefile for import
	CachefileFlag bool

	// DevPath is directory where pool devices resides
	DevPath string
}

// RunnerVar the runner variable for executing binaries.
var RunnerVar util.Runner

// GetDevPathIfNotSlashDev gets the path from given deviceID if its not prefix
// to "/dev"
func GetDevPathIfNotSlashDev(devID string) string {
	if len(devID) == 0 {
		return ""
	}

	if strings.HasPrefix(devID, "/dev") {
		return ""
	}
	lastindex := strings.LastIndexByte(devID, '/')
	if lastindex == -1 {
		return ""
	}
	devidbytes := []rune(devID)
	return string(devidbytes[0:lastindex])
}

// GetPoolName return the pool already created.
func GetPoolName() ([]string, error) {
	GetPoolStr := []string{"get", "-Hp", "name", "-o", "name"}
	poolNameByte, err := RunnerVar.RunStdoutPipe(zpool.PoolOperator, GetPoolStr...)
	if err != nil || len(string(poolNameByte)) == 0 {
		return []string{}, err
	}
	noisyPoolName := string(poolNameByte)
	sepNoisyPoolName := strings.Split(noisyPoolName, "\n")
	var poolNames []string
	for _, poolName := range sepNoisyPoolName {
		poolName = strings.TrimSpace(poolName)
		poolNames = append(poolNames, poolName)
	}
	return poolNames, nil
}

// CheckForZreplInitial is blocking call for checking status of zrepl in cstor-pool container.
func CheckForZreplInitial(ZreplRetryInterval time.Duration) {
	zStatusCmd := zfs.NewPoolStatus()
	for {
		_, err := zStatusCmd.Execute()
		if err != nil {
			time.Sleep(ZreplRetryInterval)
			klog.Errorf("zpool status returned error in zrepl startup : %v", err)
			klog.Infof("Waiting for pool container to start...")
			continue
		}
		klog.V(4).Infof("Zrepl process inside pool Container started")
		break
	}
}
