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

package cspccache

import (
	"context"
	"reflect"

	"github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/cstor-operators/tests/pkg/infra"
	"github.com/openebs/cstor-operators/tests/pkg/k8sclient"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// CSPCResourceCache holds information about Kubernetes
// resources that can be used to build a CSPC spec.
// NOTE: This cache is not thread-safe
// NOTE: CSPCResourceCache does not cache master k8s node
// and hence CSPC built using this cache will never have pool
// spec for master node. But the code can be easily customized
// to consider master node too.
type CSPCResourceCache struct {
	// NodeList is the list of the name of the nodes
	NodeList []string
	// NodeLabels is key-value map where key is node name and value
	// is again a map that holds all the labels of the node.
	NodeLabels map[string]map[string]string
	// NodeDisk is a key-vlaue map where key is the node name and
	// value is the list of disks attached to the node.
	NodeDisk map[string][]string
}

// newResourceCache return a new instance of CSPCResourceCache.
func newResourceCache() *CSPCResourceCache {
	return &CSPCResourceCache{
		NodeList:   make([]string, 0),
		NodeLabels: make(map[string]map[string]string),
		NodeDisk:   make(map[string][]string),
	}
}

// NewCSPCCache is the constructor for the CSPCResourceCache that builds the cache.
func NewCSPCCache(client *k8sclient.Client, infrastructure *infra.Infrastructure) *CSPCResourceCache {
	klog.Infof("Building CSPC Resource Cache...")
	cache := newResourceCache()
	nodeList, err := client.KubeClientSet.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		klog.Fatalf("failed to build resource cache:%s", err.Error())
	}

	for i := 0; i < len(nodeList.Items); i++ {
		// Master node can also be worker node if it is eligible for scheduling
		cache.NodeLabels[nodeList.Items[i].Name] = nodeList.Items[i].Labels
		cache.NodeList = append(cache.NodeList, nodeList.Items[i].Name)
	}

	if len(cache.NodeList) < infrastructure.NodeCount {
		klog.Fatalf("failed to build resource cache as found "+
			"only %d nodes but expected %d nodes", len(cache.NodeList), infrastructure.NodeCount)
	}

	for k, v := range cache.NodeLabels {
		// ToDo: Remove hardcoded namespace
		bdList, err := client.OpenEBSClientSet.
			OpenebsV1alpha1().
			BlockDevices("openebs").
			List(
				context.TODO(),
				v1.ListOptions{
					LabelSelector: string(types.HostNameLabelKey) +
						"=" + v[string(types.HostNameLabelKey)],
				},
			)
		if err != nil {
			klog.Fatalf("failed to build resource cache:%s", err.Error())
		}

		for _, bd := range bdList.Items {
			// If disk has filesystem then it will not participate in pool creation
			if bd.Spec.FileSystem.Type != "" {
				continue
			}
			if cache.NodeDisk[k] == nil {
				bdNameList := make([]string, 1)
				bdNameList[0] = bd.Name
				cache.NodeDisk[k] = bdNameList
			} else {
				cache.NodeDisk[k] =
					append(cache.NodeDisk[k], bd.Name)
			}
		}
	}

	return cache
}

// GetNodeNameFromLabels returns node name corresponding to the labels.
func (cc *CSPCResourceCache) GetNodeNameFromLabels(label map[string]string) string {
	nodeName := ""
	for k, v := range cc.NodeLabels {
		if reflect.DeepEqual(v, label) {
			nodeName = k
		}
	}

	if nodeName == "" {
		klog.Fatalf("Got empty node name for labels:%v", label)
	}

	return nodeName
}
