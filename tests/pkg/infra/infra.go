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

package infra

import "flag"

var (
	// NodeCount is number of storage nodes in a k8s cluster
	NodeCount int
)

// ParseFlags gets the flag values at run time
func ParseFlags() {
	flag.IntVar(&NodeCount, "nodecount", 1, "number of storage nodes to perform testing on")
}

// Infrastructure holds the details about the k8s
// infra where OpenEBS can be tested.
type Infrastructure struct {
	// NodeCount is the number of nodes
	// in the k8s infra that is capable of provisioning
	// cStor pools.
	NodeCount int
}

// NewInfrastructure return a new infrastructure instance
// by setting NodeCount.
func NewInfrastructure() *Infrastructure {
	i := &Infrastructure{}
	i.WithNodeCount(NodeCount)
	return i
}

// WithNodeCount sets the NodeCount.
func (i *Infrastructure) WithNodeCount(nodeCount int) *Infrastructure {
	i.NodeCount = nodeCount
	return i
}
