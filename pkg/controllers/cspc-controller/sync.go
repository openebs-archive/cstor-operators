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
	"github.com/openebs/cstor-operators/pkg/cspc/algorithm"
	"github.com/openebs/maya/pkg/version"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"reflect"
)

func (c *Controller) sync(cspc *cstor.CStorPoolCluster, cspiList *cstor.CStorPoolInstanceList) error {

	// If CSPC is deleted -- delete all the associated CSPI resources.
	// Cleaning up CSPI resources in case of removing poolSpec from CSPC
	// or manual CSPI deletion
	if cspc.DeletionTimestamp.IsZero() {
		err := c.cleanupCSPIResources(cspc)
		if err != nil {
			message := fmt.Sprintf("Could not sync CSPC: {%s}", err.Error())
			c.recorder.Event(cspc, corev1.EventTypeWarning, "Pool Cleanup", message)
			klog.Errorf("Could not sync CSPC %s in namesapce %s: {%s}", cspc.Name, cspc.Namespace, err.Error())
			return nil
		}
	}

	cspcGot, err := c.populateVersion(cspc)
	if err != nil {
		klog.Errorf("failed to add versionDetails to CSPC %s in namesapce %s :{%s}", cspc.Name, cspc.Namespace, err.Error())
		return nil
	}

	// If deletion timestamp is not zero on CSPC, this means CSPC is deleted
	// and all the resources associated with cspc should be deleted.
	if !cspcGot.DeletionTimestamp.IsZero() {
		err = c.handleCSPCDeletion(cspcGot)
		if err != nil {
			klog.Errorf("Failed to sync CSPC %s in namespace %s for deletion:%s", cspc.Name, cspc.Namespace, err.Error())
		}
		return nil
	}

	// Add finalizer on CSPC
	if !cspcGot.HasFinalizer(types.CSPCFinalizer) {
		cspcGot.WithFinalizer(types.CSPCFinalizer)
		cspcGot, err = c.GetStoredCStorVersionClient().CStorPoolClusters(cspcGot.Namespace).Update(cspcGot)
		if err != nil {
			klog.Errorf("Failed to add finalizer on CSPC %s in namespaces %s :{%s}", cspc.Name, cspc.Namespace, err.Error())
			return nil
		}
	}

	ac, err := algorithm.NewBuilder().
		WithCSPC(cspcGot).
		WithNameSpace(cspcGot.Namespace).
		WithKubeClient(c.kubeclientset).
		WithOpenEBSClient(c.clientset).
		Build()

	if err != nil {
		return errors.Wrapf(err, "failed to build pool config for cspc :%s in namespace %s", cspc.Name, cspc.Namespace)
	}

	pc := NewPoolConfig().WithAlgorithmConfig(ac).WithController(c)

	// Create pools if required.
	if len(cspiList.Items) < len(cspc.Spec.Pools) {
		return pc.ScaleUp(cspc, len(cspc.Spec.Pools)-len(cspiList.Items))
	}

	if len(cspiList.Items) > len(cspc.Spec.Pools) {
		// Scale Down and return
		return pc.ScaleDown(cspc)
	}

	cspisWithoutDeployment, err := c.GetCSPIWithoutDeployment(cspc)
	if err != nil {
		// Note: CSP for which pool deployment does not exists are known as orphaned.
		message := fmt.Sprintf("Error in getting orphaned CSP :{%s}", err.Error())
		c.recorder.Event(cspc, corev1.EventTypeWarning, "Pool Create", message)
		klog.Errorf("Error in getting orphaned CSP for CSPC {%s}:{%s}", cspc.Name, err.Error())
		return nil
	}

	if len(cspisWithoutDeployment) > 0 {
		pc.createDeployForCSPList(cspc, cspisWithoutDeployment)
	}

	// sync changes to cspi from cspc e.g. tunables like toleration, resource requirements etc
	pc.syncCSPI(cspc)

	pc.handleOperations()
	return nil
}

// handleCSPCDeletion handles deletion of a CSPC resource by deleting
// the associated CSPI resource(s) to it, removing the CSPC finalizer
// on BDC(s) used and then removing the CSPC finalizer on CSPC resource
// itself.

// It is necessary that CSPC resource has the CSPC finalizer on it in order to
// execute the handler.
func (c *Controller) handleCSPCDeletion(cspc *cstor.CStorPoolCluster) error {
	err := c.deleteAssociatedCSPI(cspc)

	if err != nil {
		return errors.Wrapf(err, "failed to handle CSPC deletion")
	}

	if cspc.HasFinalizer(types.CSPCFinalizer) {
		err := c.removeCSPCFinalizer(cspc)
		if err != nil {
			return errors.Wrapf(err, "failed to handle CSPC %s deletion", cspc.Name)
		}
	}

	return nil
}

// deleteAssociatedCSPI deletes the CSPI resource(s) belonging to the given CSPC resource.
// If no CSPI resource exists for the CSPC, then a levelled info log is logged and function
// returns.
func (c *Controller) deleteAssociatedCSPI(cspc *cstor.CStorPoolCluster) error {
	err := c.GetStoredCStorVersionClient().CStorPoolInstances(cspc.Namespace).DeleteCollection(
		&metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: types.CStorPoolClusterLabelKey + "=" + cspc.Name,
		},
	)

	if k8serror.IsNotFound(err) {
		klog.V(2).Infof("Associated CSPI(s) of CSPC %s is already deleted:%s", cspc.Name, err.Error())
		return nil
	}

	if err != nil {
		return errors.Wrapf(err, "failed to delete associated CSPI(s):%s", err.Error())
	}
	klog.Infof("Associated CSPI(s) of CSPC %s deleted successfully ", cspc.Name)
	return nil
}

// removeSPCFinalizer removes CSPC finalizers on associated
// BDC and CSPI resources in correct order and CSPC object itself.
func (c *Controller) removeCSPCFinalizer(cspc *cstor.CStorPoolCluster) error {

	// clean up all cspi related resources for given cspc
	err := c.cleanupCSPIResources(cspc)
	if err != nil {
		klog.Errorf("Failed to cleanup CSPC api object %s: %s", cspc.Name, err.Error())
		return nil
	}

	cspList, err := c.GetStoredCStorVersionClient().CStorPoolInstances(cspc.Namespace).List(metav1.ListOptions{
		LabelSelector: string(types.CStorPoolClusterLabelKey) + "=" + cspc.Name,
	})

	if len(cspList.Items) > 0 {
		return errors.Wrap(err, "failed to remove CSPC finalizer on associated resources as "+
			"CSPI(s) still exists for CSPC")

	}

	cspc.RemoveFinalizer(types.CSPCFinalizer)
	_, err = c.GetStoredCStorVersionClient().CStorPoolClusters(cspc.Namespace).Update(cspc)

	if err != nil {
		return errors.Wrap(err, "failed to remove CSPC finalizer on cspc resource")
	}
	return nil
}

// populateVersion assigns VersionDetails for old cspc object and newly created
// cspc
func (c *Controller) populateVersion(cspc *cstor.CStorPoolCluster) (*cstor.CStorPoolCluster, error) {
	if cspc.VersionDetails.Status.Current == "" {
		var err error
		var v string
		var obj *cstor.CStorPoolCluster
		v, err = c.EstimateCSPCVersion(cspc)
		if err != nil {
			return nil, err
		}
		cspc.VersionDetails.Status.Current = v
		// For newly created spc Desired field will also be empty.
		cspc.VersionDetails.Desired = v
		cspc.VersionDetails.Status.DependentsUpgraded = true
		obj, err = c.GetStoredCStorVersionClient().
			CStorPoolClusters(cspc.Namespace).
			Update(cspc)

		if err != nil {
			return nil, errors.Wrapf(
				err,
				"failed to update cspc %s while adding versiondetails",
				cspc.Name,
			)
		}
		klog.Infof("Version %s added on cspc %s", v, cspc.Name)
		return obj, nil
	}
	return cspc, nil
}

// EstimateCSPCVersion returns the cspi version if any cspi is present for the cspc or
// returns the maya version as the new cspi created will be of maya version
func (c *Controller) EstimateCSPCVersion(cspc *cstor.CStorPoolCluster) (string, error) {
	cspiList, err := c.clientset.CstorV1().
		CStorPoolInstances(cspc.Namespace).
		List(
			metav1.ListOptions{
				LabelSelector: types.CStorPoolClusterLabelKey + "=" + cspc.Name,
			})
	if err != nil {
		return "", errors.Wrapf(
			err,
			"failed to get the cstorpool instance list related to cspc : %s",
			cspc.Name,
		)
	}
	if len(cspiList.Items) == 0 {
		return version.Current(), nil
	}
	return cspiList.Items[0].Labels[types.OpenEBSVersionLabelKey], nil
}

// GetCSPIWithoutDeployment gets the CSPIs for whom the pool deployment does not exists.
func (c *Controller) GetCSPIWithoutDeployment(cspc *cstor.CStorPoolCluster) ([]cstor.CStorPoolInstance, error) {
	var cspiList []cstor.CStorPoolInstance
	cspiGotList, err := c.GetStoredCStorVersionClient().CStorPoolInstances(cspc.Namespace).List(metav1.ListOptions{LabelSelector: string(types.CStorPoolClusterLabelKey) + "=" + cspc.Name})
	if err != nil {
		return nil, errors.Wrapf(err, "could not list cspi for cspc {%s}", cspc.Name)
	}
	for _, cspObj := range cspiGotList.Items {
		cspObj := cspObj
		_, err := c.kubeclientset.AppsV1().Deployments(cspc.Namespace).Get(cspObj.Name, metav1.GetOptions{})
		if k8serror.IsNotFound(err) {
			cspiList = append(cspiList, cspObj)
			continue
		}
		if err != nil {
			klog.Errorf("Could not get pool deployment for cspi {%s}", cspObj.Name)
		}
	}
	return cspiList, nil
}

// syncCSPI propagates all the required changes from cspc to respective cspi.
func (pc *PoolConfig) syncCSPI(cspc *cstor.CStorPoolCluster) error {
	cspiList, err := pc.Controller.GetCSPIListForCSPC(cspc)
	if err != nil {
		return errors.Wrapf(err, "failed to sync cspi(s) from its parent cspc %s", cspc.Name)
	}
	if len(cspiList.Items) == 0 {
		return errors.Wrapf(err, "No cspi(s) found while trying to sync cspi(s) from its parent cspc %s", cspc.Name)
	}

	for _, cspi := range cspiList.Items {
		cspi := cspi
		err := pc.syncCSPIWithCSPC(cspc, &cspi)
		if err != nil {
			klog.Errorf("failed to sync cspi %s from its parent cspc %s", cspi.Name, cspc.Name)
		}
	}
}

func (pc *PoolConfig) syncCSPIWithCSPC(cspc *cstor.CStorPoolCluster, cspi *cstor.CStorPoolInstance) error {
	cspiCopy := cspi.DeepCopy()
	klog.V(2).Infof("Syncing cspi %s from parent cspc %s", cspiCopy.Name, cspc.Name)
	for _, poolSpec := range cspc.Spec.Pools {
		poolSpec := poolSpec
		cspiCopy.Spec.PoolConfig = poolSpec.PoolConfig
		defaultPoolConfig(cspiCopy, cspc)
	}

	if !reflect.DeepEqual(cspiCopy, cspi) {
		gotCSPI, err := pc.Controller.GetStoredCStorVersionClient().CStorPoolInstances(cspiCopy.Namespace).Update(cspiCopy)
		if err != nil {
			return errors.Errorf("Failed to sync cspi %s from parent cspc %s", cspiCopy.Name, cspc.Name)
		}
		_, err = pc.Controller.kubeclientset.AppsV1().Deployments(gotCSPI.Namespace).Update(pc.AlgorithmConfig.GetPoolDeploySpec(gotCSPI))
		if err != nil {
			return errors.Errorf("Failed to sync cspi %s from parent cspc %s", cspiCopy.Name, cspc.Name)
		}
	}
	return nil
}

func defaultPoolConfig(cspi *cstor.CStorPoolInstance, cspc *cstor.CStorPoolCluster) {
	if cspi.Spec.PoolConfig.Resources == nil {
		cspi.Spec.PoolConfig.Resources = cspc.Spec.DefaultResources
	}
	if cspi.Spec.PoolConfig.AuxResources == nil {
		cspi.Spec.PoolConfig.AuxResources = cspc.Spec.DefaultAuxResources
	}
	if cspi.Spec.PoolConfig.Tolerations == nil {
		cspi.Spec.PoolConfig.Tolerations = cspc.Spec.Tolerations
	}

	if cspi.Spec.PoolConfig.PriorityClassName == "" {
		cspi.Spec.PoolConfig.PriorityClassName = cspc.Spec.DefaultPriorityClassName
	}
}
