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

package cspcspecbuilder

import (
	"reflect"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	"github.com/openebs/cstor-operators/tests/pkg/cache/cspccache"
	"github.com/openebs/cstor-operators/tests/pkg/infra"
	"k8s.io/klog"
)

// CSPCSpecBuilder is used to build CSPC spec.
// It uses CSPCCache and CSPCSpecData to help cients build efficiently and easily.
type CSPCSpecBuilder struct {
	CSPCCache    *cspccache.CSPCResourceCache
	Infra        *infra.Infrastructure
	CSPC         *cstor.CStorPoolCluster
	CSPCSpecData *CSPCSpecData
}

// CSPCSpecData is used to keep track of used and unused node and disks.
type CSPCSpecData struct {
	UsedNodes   map[string]bool
	UnUsedNodes map[string]bool
	UsedDisks   map[string]bool
	UnUsedDisk  map[string]bool
}

// NewCSPCSpecData returns an empty instance of CSPCSpeData
func NewCSPCSpecData() *CSPCSpecData {
	return &CSPCSpecData{
		UsedNodes:   map[string]bool{},
		UnUsedNodes: map[string]bool{},
		UsedDisks:   map[string]bool{},
		UnUsedDisk:  map[string]bool{},
	}
}

// AddNodeToUsedSet add a node to used set and removes from unused set.
func (cd *CSPCSpecData) AddNodeToUsedSet(nodeName string) {
	cd.UsedNodes[nodeName] = true
	delete(cd.UnUsedNodes, nodeName)
}

// AddDiskToUsedSet adds a disk to used set and removes from unusued set.
func (cd *CSPCSpecData) AddDiskToUsedSet(diskName string) {
	cd.UsedDisks[diskName] = true
	delete(cd.UnUsedDisk, diskName)
}

// AddNodeToUnusedSet adds a node to unused set and removes from used set.
func (cd *CSPCSpecData) AddNodeToUnusedSet(nodeName string) {
	cd.UnUsedNodes[nodeName] = true
	delete(cd.UsedNodes, nodeName)
}

// AddDiskToUnusedSet adds a disk to unused set and removes from used set.
func (cd *CSPCSpecData) AddDiskToUnusedSet(diskName string) {
	cd.UnUsedDisk[diskName] = true
	delete(cd.UsedDisks, diskName)
}

// NewCSPCSpecBuilder returns a new instance of CSPCSpecBuilder
func NewCSPCSpecBuilder(cspcCache *cspccache.CSPCResourceCache, infra *infra.Infrastructure) *CSPCSpecBuilder {
	// Initialize CSPCSpecData
	cspcSpecData := NewCSPCSpecData()
	for _, nodeName := range cspcCache.NodeList {
		cspcSpecData.UnUsedNodes[nodeName] = true
		for _, bd := range cspcCache.NodeDisk[nodeName] {
			cspcSpecData.UnUsedDisk[bd] = true
		}
	}
	return &CSPCSpecBuilder{
		CSPCCache:    cspcCache,
		CSPCSpecData: cspcSpecData,
		Infra:        infra,
	}
}

type ReplacementTracer struct {
	OldBD    string
	NewBD    string
	Replaced bool
}

func NewReplacementTracer() *ReplacementTracer {
	return &ReplacementTracer{}
}

// ReplaceBlockDevice replaces a block device at the provided position in the CSPC
func (c *CSPCSpecBuilder) ReplaceBlockDeviceAtPos(poolSpecPos, raidGroupPos, bdPos int, rt *ReplacementTracer) *CSPCSpecBuilder {
	oldBD := c.CSPC.Spec.Pools[poolSpecPos].DataRaidGroups[raidGroupPos].CStorPoolInstanceBlockDevices[bdPos].BlockDeviceName
	nodeName := c.CSPCCache.GetNodeNameFromLabels(c.CSPC.Spec.Pools[poolSpecPos].NodeSelector)
	bdList := c.CSPCCache.NodeDisk[nodeName]
	newBD := ""
	for _, v := range bdList {
		if c.CSPCSpecData.UnUsedDisk[v] {
			newBD = v
			break
		}
	}

	if newBD == "" {
		klog.Fatalf("Could not find a new block device for replacement")
	}
	c.CSPC.Spec.Pools[poolSpecPos].DataRaidGroups[raidGroupPos].
		CStorPoolInstanceBlockDevices[bdPos].BlockDeviceName = newBD

	rt.NewBD = newBD
	rt.OldBD = oldBD
	rt.Replaced = true

	c.CSPCSpecData.AddDiskToUsedSet(newBD)
	c.CSPCSpecData.AddDiskToUnusedSet(oldBD)
	return c
}

// ReplaceBlockDevice replaces given oldBd with the given newBD
func (c *CSPCSpecBuilder) ReplaceBlockDevice(oldBD, newBD string) *CSPCSpecBuilder {
	replaced := false
	for i := 0; i < len(c.CSPC.Spec.Pools); i++ {
		for j := 0; j < len(c.CSPC.Spec.Pools[i].DataRaidGroups); j++ {
			for k := 0; k < len(c.CSPC.Spec.Pools[i].DataRaidGroups[j].CStorPoolInstanceBlockDevices); k++ {
				if c.CSPC.Spec.Pools[i].DataRaidGroups[j].CStorPoolInstanceBlockDevices[k].BlockDeviceName == oldBD {
					c.CSPC.Spec.Pools[i].DataRaidGroups[j].CStorPoolInstanceBlockDevices[k].BlockDeviceName = newBD
					replaced = true
					break
				}
			}
		}
	}
	if !replaced {
		klog.Fatalf("Could not find a %s block device for replacement", oldBD)
	}
	c.CSPCSpecData.AddDiskToUsedSet(newBD)
	c.CSPCSpecData.AddDiskToUnusedSet(oldBD)
	return c
}

// RemovePoolSpec removes a pool spec from the CSPC.
// It always removes the last pool spec
func (c *CSPCSpecBuilder) RemovePoolSpec() *CSPCSpecBuilder {
	currentSize := len(c.CSPC.Spec.Pools)
	if currentSize < 1 {
		klog.Warning("Could not remove pool spec as no pool spec found in cspc")
	} else {
		nodeToBeRemoved := ""
		for k, v := range c.CSPCCache.NodeLabels {
			if reflect.DeepEqual(v, c.CSPC.Spec.Pools[currentSize-1].NodeSelector) {
				nodeToBeRemoved = k
			}
		}
		c.CSPCSpecData.UnUsedNodes[nodeToBeRemoved] = true
		delete(c.CSPCSpecData.UsedNodes, nodeToBeRemoved)
		c.CSPC.Spec.Pools = c.CSPC.Spec.Pools[:currentSize-1]
	}
	return c
}

// AddPoolSpec adds a pool spec in the CSPC in accordance with the provided
// arguments.
func (c *CSPCSpecBuilder) AddPoolSpec(nodeName string, poolType string, bdCount int) *CSPCSpecBuilder {
	if nodeName == "" {
		klog.Fatal("Got empty node name while adding a pool spec")
	}
	c.CSPCSpecData.AddNodeToUsedSet(nodeName)
	nodeSelector := c.CSPCCache.NodeLabels[nodeName]
	blockDevices := make([]cstor.CStorPoolInstanceBlockDevice, 0)

	if len(c.CSPCCache.NodeDisk[nodeName]) < bdCount {
		klog.Fatalf("Not enough block "+
			"devices available for node %s: want %d,got %s",
			nodeName, bdCount, c.CSPCCache.NodeDisk[nodeName])
	}

	for i := 0; i < bdCount; i++ {
		newBlockDevice := cstor.CStorPoolInstanceBlockDevice{
			BlockDeviceName: c.CSPCCache.NodeDisk[nodeName][i],
		}
		blockDevices = append(blockDevices, newBlockDevice)
		c.CSPCSpecData.AddDiskToUsedSet(newBlockDevice.BlockDeviceName)
	}

	newPoolSpec := cstor.NewPoolSpec().
		WithNodeSelector(nodeSelector).
		WithDataRaidGroups(
			*cstor.NewRaidGroup().
				WithCStorPoolInstanceBlockDevices(
					blockDevices...,
				),
		).
		WithPoolConfig(*cstor.NewPoolConfig().WithDataRaidGroupType(poolType))

	c.CSPC.WithPoolSpecs([]cstor.PoolSpec{*newPoolSpec}...)
	return c
}

// BuildCSPC builds a CSPC spec in accordance with the provided arguments
func (c *CSPCSpecBuilder) BuildCSPC(cspcName, namespace, poolType string, bdCount, poolCount int) *CSPCSpecBuilder {
	cspc := cstor.NewCStorPoolCluster().
		WithName(cspcName).
		WithNamespace(namespace)

	c.CSPC = cspc

	currentPoolCount := 0
	index := 0
	for currentPoolCount < poolCount && index < len(c.CSPCCache.NodeList) {
		// Pick the first node
		nodeName := c.CSPCCache.NodeList[index]
		if c.CSPCSpecData.UnUsedNodes[nodeName] {
			c.AddPoolSpec(nodeName, poolType, bdCount)
			currentPoolCount++
		}
		index++
	}

	if currentPoolCount != poolCount {
		klog.Fatalf("failed to build cspc for %d pool count "+
			"as only %d nodes were available", poolCount, currentPoolCount)
	}

	return c
}

// GetCSPCSpec returns the CSPC spec from the spec builder.
func (c *CSPCSpecBuilder) GetCSPCSpec() *cstor.CStorPoolCluster {
	return c.CSPC
}

// SetCSPCSpec sets the CSPC spec in spec builder.
func (c *CSPCSpecBuilder) SetCSPCSpec(cspc *cstor.CStorPoolCluster) {
	c.CSPC = cspc
}

// ResetCSPCSpecData clears the CSPCSpecData
func (c *CSPCSpecBuilder) ResetCSPCSpecData() {
	c.CSPCSpecData = NewCSPCSpecData()
}
