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

package cvcspecbuilder

import (
	cstorapis "github.com/openebs/api/pkg/apis/cstor/v1"
	"github.com/openebs/cstor-operators/tests/pkg/infra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// CVCSpecBuilder is used to build/update CVC spec.
// It uses CVCSpecData to help cients build efficiently and easily.
type CVCSpecBuilder struct {
	Infra       *infra.Infrastructure
	CVC         *cstorapis.CStorVolumeConfig
	CVCSpecData *CVCSpecData
}

// CVCSpecData is used to keep track of used and unused pools.
type CVCSpecData struct {
	UsedPools   map[string]bool
	UnUsedPools map[string]bool
}

// NewCVCSpecData returns an empty instance of CVCSpeData
func NewCVCSpecData() *CVCSpecData {
	return &CVCSpecData{
		UsedPools:   map[string]bool{},
		UnUsedPools: map[string]bool{},
	}
}

// NewCVCSpecBuilder returns an emty instance of CVCSpecBuilder
func NewCVCSpecBuilder(infra *infra.Infrastructure, poolNames []string) *CVCSpecBuilder {
	cvcSpecData := NewCVCSpecData()
	for _, name := range poolNames {
		cvcSpecData.UnUsedPools[name] = true
	}
	return &CVCSpecBuilder{
		Infra:       infra,
		CVCSpecData: cvcSpecData,
	}
}

// AddPoolToUsedSet add a pool to used set and removes from unused set.
func (cd *CVCSpecData) AddPoolToUsedSet(poolName string) {
	cd.UsedPools[poolName] = true
	delete(cd.UnUsedPools, poolName)
}

// AddPoolToUnusedSet adds a pool to unused set and removes from used set.
func (cd *CVCSpecData) AddPoolToUnusedSet(poolName string) {
	cd.UnUsedPools[poolName] = true
	delete(cd.UsedPools, poolName)
}

// GetUnusedPoolNames returns list of unused pool names for volume provisioning
func (cd *CVCSpecData) GetUnusedPoolNames() []string {
	unUsedPoolNames := make([]string, 1)
	for name := range cd.UnUsedPools {
		unUsedPoolNames = append(unUsedPoolNames, name)
	}
	return unUsedPoolNames
}

// SetCVCSpec sets the CVC spec in spec builder.
func (c *CVCSpecBuilder) SetCVCSpec(cvc *cstorapis.CStorVolumeConfig) {
	c.addVolumeReplicaPoolsToUsedSet(cvc)
	c.CVC = cvc
}

// GetCVCSpec sets the CVC spec in spec builder.
func (c *CVCSpecBuilder) GetCVCSpec() *cstorapis.CStorVolumeConfig {
	return c.CVC
}

// addVolumeReplicaPoolsToUsedSet will adds the list of used pools into used set
func (c *CVCSpecBuilder) addVolumeReplicaPoolsToUsedSet(cvc *cstorapis.CStorVolumeConfig) {
	for _, replicaInfo := range cvc.Spec.Policy.ReplicaPoolInfo {
		if c.CVCSpecData.UnUsedPools[replicaInfo.PoolName] {
			c.CVCSpecData.AddPoolToUsedSet(replicaInfo.PoolName)
		}
	}
}

// ScaleupCVC will scale the volume replicas
func (c *CVCSpecBuilder) ScaleupCVC(poolNames []string) *CVCSpecBuilder {
	if len(poolNames)+len(c.CVC.Spec.Policy.ReplicaPoolInfo) > 5 {
		klog.Fatalf("OpenEBS doesn't support more than 5 copies of data")
	}

	for _, poolName := range poolNames {
		if c.CVCSpecData.UnUsedPools[poolName] {
			c.CVCSpecData.AddPoolToUsedSet(poolName)
			c.CVC.Spec.Policy.ReplicaPoolInfo = append(
				c.CVC.Spec.Policy.ReplicaPoolInfo, cstorapis.ReplicaPoolInfo{PoolName: poolName})
		}
	}
	return c
}

// SetResourceLimits sets the resource limits for target deployment
func (c *CVCSpecBuilder) SetResourceLimits(auxResourceLimits, resourceLimits *corev1.ResourceRequirements) *CVCSpecBuilder {
	c.CVC.Spec.Policy.Target.AuxResources = auxResourceLimits
	c.CVC.Spec.Policy.Target.Resources = resourceLimits
	return c
}
