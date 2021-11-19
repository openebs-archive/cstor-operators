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

package cspccontroller

import (
	"context"
	"fmt"

	"github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/cstor-operators/pkg/cspc/algorithm"

	"time"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// PoolConfig embeds alogrithm config from algorithm package and Controller object.
type PoolConfig struct {
	AlgorithmConfig *algorithm.Config
	Controller      *Controller
}

// NewPoolConfig returns an empty instance of poolconfig object.
func NewPoolConfig() *PoolConfig {
	return &PoolConfig{}
}

// WithAlgorithmConfig sets the AlgorithmConfig field of the poolconfig object.
func (pc *PoolConfig) WithAlgorithmConfig(ac *algorithm.Config) *PoolConfig {
	pc.AlgorithmConfig = ac
	return pc
}

// WithController sets the controller field of the poolconfig object.
func (pc *PoolConfig) WithController(c *Controller) *PoolConfig {
	pc.Controller = c
	return pc
}

// syncCSPC compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the cspc resource
// with the current status of the resource.
func (c *Controller) syncCSPC(key string) error {
	startTime := time.Now()
	klog.V(4).Infof("Started syncing cstorpoolcluster %q (%v)", key, startTime)
	defer func() {
		klog.V(4).Infof("Finished syncing cstorpoolcluster %q (%v)", key, time.Since(startTime))
	}()

	// The sync will also execute after every RESYNC_INTERVAL even there is no change in
	// the CR object.
	// The RESYNC_INTERVAL time period is get by getSyncInterval() function defined
	// in cstor-operator/cspc-operator/app/start.go

	// Convert the namespace/name string into a distinct namespace and name
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	// Get the cspc resource with this namespace/name
	cspc, err := c.cspcLister.CStorPoolClusters(ns).Get(name)
	if k8serror.IsNotFound(err) {
		runtime.HandleError(fmt.Errorf("cspc '%s' has been deleted", key))
		return nil
	}
	if err != nil {
		return err
	}

	// Deep-copy otherwise we are mutating our cache.
	cspcGot := cspc.DeepCopy()
	cspiList, _ := c.GetCSPIListForCSPC(cspcGot)
	err = c.sync(cspcGot, cspiList)
	return err
}

func (c *Controller) enqueue(cspc *cstor.CStorPoolCluster) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(cspc); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// GetCSPIListForCSPC returns list of cspi parented by cspc.
func (c *Controller) GetCSPIListForCSPC(cspc *cstor.CStorPoolCluster) (*cstor.CStorPoolInstanceList, error) {
	return c.GetStoredCStorVersionClient().
		CStorPoolInstances(cspc.Namespace).
		List(context.TODO(),
			metav1.ListOptions{
				LabelSelector: types.CStorPoolClusterLabelKey + "=" + cspc.Name,
			})
}
