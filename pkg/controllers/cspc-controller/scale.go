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
	"fmt"
	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	"github.com/openebs/api/pkg/apis/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog"

	"github.com/pkg/errors"
)

// ScaleUp creates as many cstor pool on a node as pendingPoolCount.
func (pc *PoolConfig) ScaleUp(cspc *cstor.CStorPoolCluster, pendingPoolCount int) {
	needsStatusUpdate := false
	for poolCount := 1; poolCount <= pendingPoolCount; poolCount++ {
		err := pc.CreateCSPI(cspc)
		if err != nil {
			message := fmt.Sprintf("Pool provisioning failed for %d/%d ", poolCount, pendingPoolCount)
			pc.Controller.recorder.Event(cspc, corev1.EventTypeWarning, "Create", message)
			runtime.HandleError(errors.Wrapf(err, "Pool provisioning failed for %d/%d for cstorpoolcluster %s", poolCount, pendingPoolCount, cspc.Name))
		} else {
			needsStatusUpdate = true
			message := fmt.Sprintf("Pool Provisioned %d/%d ", poolCount, pendingPoolCount)
			pc.Controller.recorder.Event(cspc, corev1.EventTypeNormal, "Create", message)
			klog.Infof("Pool provisioned successfully %d/%d for cstorpoolcluster %s", poolCount, pendingPoolCount, cspc.Name)
		}
	}
	if needsStatusUpdate {
		err := pc.Controller.UpdateStatus(cspc)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to update cspc %s status", cspc.Name))
		}
	}
}

// CreateCSPI creates CSPI
func (pc *PoolConfig) CreateCSPI(cspc *cstor.CStorPoolCluster) error {
	cspi, err := pc.AlgorithmConfig.GetCSPISpec()
	if err != nil {
		return err
	}
	// The cpsi variable is written back here, This is important as cspi uid is passed to pool deployment
	// The uid does not exist before cspi creation.
	cspi, err = pc.Controller.GetStoredCStorVersionClient().CStorPoolInstances(cspc.Namespace).Create(cspi)

	if err != nil {
		return err
	}
	err = pc.CreateCSPIDeployment(cspc, cspi)
	if err != nil {
		return err
	}
	return nil
}

func (pc *PoolConfig) createDeployForCSPList(cspc *cstor.CStorPoolCluster, cspList []cstor.CStorPoolInstance) {
	for _, cspObj := range cspList {
		cspObj := cspObj
		err := pc.CreateCSPIDeployment(cspc, &cspObj)
		if err != nil {
			message := fmt.Sprintf("Failed to create pool deployment for CSP %s: %s", cspObj.Name, err.Error())
			pc.Controller.recorder.Event(cspc, corev1.EventTypeWarning, "PoolDeploymentCreate", message)
			runtime.HandleError(errors.Errorf("Failed to create pool deployment for CSP %s: %s", cspObj.Name, err.Error()))
		}
	}
}

// CreateStoragePool creates the required resource to provision a cStor pool
func (pc *PoolConfig) CreateCSPIDeployment(cspc *cstor.CStorPoolCluster, cspi *cstor.CStorPoolInstance) error {
	deploy := pc.AlgorithmConfig.GetPoolDeploySpec(cspi)
	_, err := pc.Controller.kubeclientset.AppsV1().Deployments(cspi.Namespace).Create(deploy)
	if err != nil {
		return err
	}
	return nil
}

// DownScalePool deletes the required pool.
func (pc *PoolConfig) ScaleDown(cspc *cstor.CStorPoolCluster) {
	needsStatusUpdate := false
	orphanedCSP, err := pc.getOrphanedCStorPools(cspc)

	if err != nil {
		pc.Controller.recorder.Event(cspc, corev1.EventTypeWarning,
			"DownScale", "Pool downscale failed "+err.Error())
		klog.Errorf("Pool downscale failed as could not get orphaned CSP(s):{%s}" + err.Error())
		return
	}

	for _, cspiName := range orphanedCSP {
		pc.Controller.recorder.Event(cspc, corev1.EventTypeNormal,
			"DownScale", "De-provisioning pool "+cspiName)

		// TODO : As part of deleting a CSP, do we need to delete associated BDCs ?
		needsStatusUpdate = true
		err := pc.Controller.GetStoredCStorVersionClient().CStorPoolInstances(cspc.Namespace).Delete(cspiName, &metav1.DeleteOptions{})
		if err != nil {
			pc.Controller.recorder.Event(cspc, corev1.EventTypeWarning,
				"DownScale", "De-provisioning pool "+cspiName+"failed")
			klog.Errorf("De-provisioning pool %s failed: %s", cspiName, err)
		}
	}

	if needsStatusUpdate {
		err := pc.Controller.UpdateStatus(cspc)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to update cspc %s status", cspc.Name))
		}
	}
}

// getOrphanedCStorPools returns a list of CSPI names that should be deleted.
func (pc *PoolConfig) getOrphanedCStorPools(cspc *cstor.CStorPoolCluster) ([]string, error) {
	var orphanedCSP []string
	nodePresentOnCSPC, err := pc.getNodePresentOnCSPC(cspc)
	if err != nil {
		return []string{}, errors.Wrap(err, "could not get node names of pool config present on CSPC")
	}
	cspList, err := pc.Controller.GetStoredCStorVersionClient().CStorPoolInstances(cspc.Namespace).List(
		metav1.ListOptions{LabelSelector: string(types.CStorPoolClusterLabelKey) + "=" + cspc.Name})

	if err != nil {
		return []string{}, errors.Wrap(err, "could not list CSP(s)")
	}

	for _, cspObj := range cspList.Items {
		cspObj := cspObj
		if nodePresentOnCSPC[cspObj.Spec.HostName] {
			continue
		}
		orphanedCSP = append(orphanedCSP, cspObj.Name)
	}
	return orphanedCSP, nil
}

// getNodePresentOnCSPC returns a map of node names where pool should
// be present.
func (pc *PoolConfig) getNodePresentOnCSPC(cspc *cstor.CStorPoolCluster) (map[string]bool, error) {
	nodeMap := make(map[string]bool)
	for _, pool := range cspc.Spec.Pools {
		nodeName, err := pc.AlgorithmConfig.GetNodeFromLabelSelector(pool.NodeSelector)
		if err != nil {
			return nil, errors.Wrapf(err,
				"could not get node name for node selector {%v} "+
					"from cspc %s", pool.NodeSelector, cspc.Name)
		}
		nodeMap[nodeName] = true
	}
	return nodeMap, nil
}
