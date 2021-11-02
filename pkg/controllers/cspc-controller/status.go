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

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/cstor-operators/pkg/controllers/cspc-controller/util"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"time"

	"k8s.io/klog"
)

func (c *Controller) UpdateStatusEventually(cspc *cstor.CStorPoolCluster) error {
	maxRetry := 3
	err := c.UpdateStatus(cspc)

	if err != nil {
		klog.Errorf("failed to update cspc %s status: will retry %d times at 2s interval: {%s}",
			cspc.Name, maxRetry, err.Error())
		for maxRetry > 0 {
			cspcNew, err := c.GetStoredCStorVersionClient().
				CStorPoolClusters(cspc.Namespace).
				Get(context.TODO(), cspc.Name, metav1.GetOptions{})

			if err != nil {
				// this is possible due to etcd unavailability so do not retry more here
				return errors.Wrapf(err, "failed to update cspc status")
			}

			err = c.UpdateStatus(cspcNew)
			if err != nil {
				maxRetry = maxRetry - 1
				klog.Errorf("failed to update cspc %s status: will retry %d times at 2s interval : {%s}",
					cspc.Name, maxRetry, err.Error())
				time.Sleep(2 * time.Second)
				continue
			}
			return nil
		}

	}
	return err
}

func (c *Controller) UpdateStatus(cspc *cstor.CStorPoolCluster) error {
	status, err := c.calculateStatus(cspc)
	if err != nil {
		return errors.Wrapf(err, "failed to calculate cspc %s status", cspc.Name)
	}
	cspc.Status = status
	_, err = c.GetStoredCStorVersionClient().CStorPoolClusters(cspc.Namespace).Update(context.TODO(), cspc, metav1.UpdateOptions{})

	if err != nil {
		return errors.Wrapf(err, "failed to update cspc %s in namespace %s", cspc.Name, cspc.Namespace)
	}

	return nil
}

func (c *Controller) calculateStatus(cspc *cstor.CStorPoolCluster) (cstor.CStorPoolClusterStatus, error) {
	var healthyCSPIs int32
	cspiList, err := c.GetCSPIListForCSPC(cspc)
	if err != nil {
		return cstor.CStorPoolClusterStatus{}, errors.Wrapf(err, "failed to list cspi(s) for cspc %s in namespace %s", cspc.Name, cspc.Namespace)
	}

	// List all corresponding pool managers for the cspc
	poolManagerList, err := c.kubeclientset.
		AppsV1().
		Deployments(cspc.Namespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: string(types.CStorPoolClusterLabelKey) + "=" + cspc.Name})
	if err != nil {
		return cstor.CStorPoolClusterStatus{}, errors.Wrapf(err, "failed to list pool-manager deployments for cspc %s in namespace %s", cspc.Name, cspc.Namespace)
	}

	cspiNameToPoolManager := make(map[string]appsv1.Deployment)

	for _, poolmanager := range poolManagerList.Items {
		poolmanager := poolmanager // pin it
		// note: name of cspi and corresponding pool manager is same.
		cspiNameToPoolManager[poolmanager.Name] = poolmanager
	}

	status := cstor.CStorPoolClusterStatus{
		ProvisionedInstances: int32(len(cspiList.Items)),
		DesiredInstances:     int32(len(cspc.Spec.Pools)),
	}

	// Copy conditions one by one so we won't mutate the original object.
	conditions := cspc.Status.Conditions
	for i := range conditions {
		status.Conditions = append(status.Conditions, conditions[i])
	}

	unavailablePoolManagers := ""
	isPoolManagerUnavailable := false

	for _, cspi := range cspiList.Items {
		if IsPoolMangerAvailable(cspiNameToPoolManager[cspi.Name]) {
			if cspi.Status.Phase == cstor.CStorPoolStatusOnline {
				healthyCSPIs++
			}
		} else {
			isPoolManagerUnavailable = true
			unavailablePoolManagers = unavailablePoolManagers + cspi.Name + " "
		}
	}

	status.HealthyInstances = healthyCSPIs

	if isPoolManagerUnavailable {
		minAvailability := util.NewCSPCCondition(util.PoolManagerAvailable, v1.ConditionFalse,
			util.MinimumPoolManagersUnAvailable, "Pool manager(s): "+unavailablePoolManagers+":does not have minimum available pod")
		util.SetCSPCCondition(&status, *minAvailability)
	} else {
		minAvailability := util.NewCSPCCondition(util.PoolManagerAvailable, v1.ConditionTrue,
			util.MinimumPoolManagersAvailable, "Pool manager(s) have minimum available pod")
		util.SetCSPCCondition(&status, *minAvailability)
	}

	return status, nil
}

func IsPoolMangerAvailable(pm appsv1.Deployment) bool {
	return pm.Status.ReadyReplicas >= 1
}
