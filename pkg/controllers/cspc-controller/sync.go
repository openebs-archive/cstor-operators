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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/openebs/cstor-operators/pkg/cspc/algorithm"
	"github.com/openebs/cstor-operators/pkg/version"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/klog"
)

type upgradeParams struct {
	cspc   *cstor.CStorPoolCluster
	client clientset.Interface
}

type upgradeFunc func(u *upgradeParams) (*cstor.CStorPoolCluster, error)

var (
	upgradeMap = map[string]upgradeFunc{}
	// defaultROThresholdLimit is the default
	// value in form of percentage for ROThreshold limit
	defaultROThresholdLimit = 85
)

func (c *Controller) sync(cspc *cstor.CStorPoolCluster, cspiList *cstor.CStorPoolInstanceList) error {

	// If deletion timestamp is not zero on CSPC, this means CSPC is deleted
	// and all the resources associated with cspc should be deleted.
	if !cspc.DeletionTimestamp.IsZero() {
		err := c.handleCSPCDeletion(cspc)
		if err != nil {
			message := fmt.Sprintf("Could not sync for CSPC:{%s} deletion", err.Error())
			c.recorder.Event(cspc, corev1.EventTypeWarning, "CSPC Cleanup", message)
			klog.Errorf("Failed to cleanup CSPC %s in namespace %s: %s", cspc.Name, cspc.Namespace, err.Error())
		}
		return nil
	}

	// cleaning up CSPI resources in case of removing poolSpec from CSPC
	// or manual CSPI deletion
	// This should be performed before reconcileVersion is done.
	if cspc.DeletionTimestamp.IsZero() {
		cspiList, err := c.GetCSPIListForCSPC(cspc)
		if err != nil {
			message := fmt.Sprintf("Could not sync CSPC: {%s}", err.Error())
			c.recorder.Event(cspc, corev1.EventTypeWarning, "Pool Cleanup", message)
			klog.Errorf("Could not sync CSPC %s in namespace %s: {%s}", cspc.Name, cspc.Namespace, err.Error())
			return nil
		}

		err = c.cleanupCSPIResources(cspiList)
		if err != nil {
			message := fmt.Sprintf("Could not sync CSPC: {%s}", err.Error())
			c.recorder.Event(cspc, corev1.EventTypeWarning, "Pool Cleanup", message)
			klog.Errorf("Could not sync CSPC %s in namespace %s: {%s}", cspc.Name, cspc.Namespace, err.Error())
			return nil
		}
	}

	cspcGot, err := c.populateVersion(cspc)
	if err != nil {
		klog.Errorf("failed to add versionDetails to CSPC %s in namespace %s :{%s}", cspc.Name, cspc.Namespace, err.Error())
		return nil
	}

	cspcGot, err = c.reconcileVersion(cspcGot)
	if err != nil {
		message := fmt.Sprintf("Failed to upgrade cspc to %s version: %s",
			cspcGot.VersionDetails.Desired,
			err.Error())
		klog.Errorf("failed to upgrade cspc %s:%s", cspcGot.Name, err.Error())
		c.recorder.Event(cspc, corev1.EventTypeWarning, "FailedUpgrade", message)
		cspcGot.VersionDetails.Status.SetErrorStatus(
			"Failed to reconcile cspc version",
			err,
		)
		_, err = c.clientset.CstorV1().CStorPoolClusters(cspcGot.Namespace).Update(context.TODO(), cspcGot, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update versionDetails status for cspc %s:%s", cspcGot.Name, err.Error())
		}
		return nil
	}

	if ok, reason := c.ShouldReconcile(*cspc); !ok {
		// Do not reconcile this cspc
		message := fmt.Sprintf("Can not reconcile CSPC %s as %s", cspc.Name, reason)
		c.recorder.Event(cspc, corev1.EventTypeWarning, "CSPC Reconcile", message)
		klog.Warningf("Can not reconcile CSPC %s in namespace %s as %s", cspc.Name, cspc.Namespace, reason)
		return nil
	}

	cspcGot, err = c.populateDesiredInstances(cspcGot)
	if err != nil {
		klog.Errorf("failed to add desired instances to CSPC %s in namespace %s :{%s}", cspc.Name, cspc.Namespace, err.Error())
		return nil
	}

	// Add finalizer on CSPC
	if !cspcGot.HasFinalizer(types.CSPCFinalizer) {
		cspcGot.WithFinalizer(types.CSPCFinalizer)
		cspcGot, err = c.GetStoredCStorVersionClient().CStorPoolClusters(cspcGot.Namespace).Update(context.TODO(), cspcGot, metav1.UpdateOptions{})
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
	if len(cspiList.Items) < len(cspcGot.Spec.Pools) {
		pc.ScaleUp(cspcGot, len(cspcGot.Spec.Pools)-len(cspiList.Items))
	} else if len(cspiList.Items) > len(cspcGot.Spec.Pools) {
		// Scale Down pools if required
		pc.ScaleDown(cspcGot)
	}

	cspisWithoutDeployment, err := c.GetCSPIWithoutDeployment(cspcGot)
	if err != nil {
		// Note: CSP for which pool deployment does not exists are known as orphaned.
		message := fmt.Sprintf("Error in getting orphaned CSP :{%s}", err.Error())
		c.recorder.Event(cspcGot, corev1.EventTypeWarning, "Pool Create", message)
		klog.Errorf("Error in getting orphaned CSP for cspcGot {%s}:{%s}", cspcGot.Name, err.Error())
	}

	if len(cspisWithoutDeployment) > 0 {
		pc.createDeployForCSPList(cspcGot, cspisWithoutDeployment)
	}

	// sync changes to cspi from cspc e.g. tunables like toleration, resource requirements etc
	err = pc.syncCSPI(cspcGot)

	// Not returning error so that `handleOperations` can also be executed.
	if err != nil {
		klog.Errorf("failed to sync cspi(s) of cspc %s", cspcGot.Name)
	}

	pc.handleOperations()

	err = c.UpdateStatusEventually(cspcGot)
	if err != nil {
		message := fmt.Sprintf("Error in updating status:{%s}", err.Error())
		c.recorder.Event(cspcGot, corev1.EventTypeWarning, "Status Update", message)
		klog.Errorf("Error in updating  CSPC %s status:{%s}", cspcGot.Name, err.Error())
		return nil
	}

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
		return errors.Wrap(err, "failed to delete associated cspi(s)")
	}

	if cspc.HasFinalizer(types.CSPCFinalizer) {
		err := c.removeCSPCFinalizer(cspc)
		if err != nil {
			return errors.Wrap(err, "failed to remove cspc finalizers on cspi objects")
		}
	}

	return nil
}

// deleteAssociatedCSPI deletes the CSPI resource(s) belonging to the given CSPC resource.
// If no CSPI resource exists for the CSPC, then a levelled info log is logged and function
// returns.
func (c *Controller) deleteAssociatedCSPI(cspc *cstor.CStorPoolCluster) error {
	err := c.GetStoredCStorVersionClient().CStorPoolInstances(cspc.Namespace).DeleteCollection(
		context.TODO(),
		metav1.DeleteOptions{},
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

	cspiList, err := c.GetCSPIListForCSPC(cspc)
	if err != nil {
		return errors.Wrapf(err, "could not list cspi(s)")
	}

	// clean up all cspi related resources for given cspc
	err = c.cleanupCSPIResources(cspiList)
	if err != nil {
		return errors.Wrap(err, "failed to cleanup cspc")
	}

	cspList, err := c.GetCSPIListForCSPC(cspc)
	if err != nil {
		return errors.Wrapf(err, "could not list cspi(s)")
	}

	if len(cspList.Items) > 0 {
		return errors.Wrap(err, "failed to remove CSPC finalizer on associated resources as "+
			"CSPI(s) still exists for CSPC")
	}

	// If the BD(s) are claimed and the CSPI(s) do not spawn up for some reason,
	// the pending BDC(s) finalizer don't get removed automatically.
	// The below code handles cleanup for such cases.
	bdcList, err := c.GetStoredOpenebsVersionClient().BlockDeviceClaims(cspc.Namespace).List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: string(types.CStorPoolClusterLabelKey) + "=" + cspc.Name,
		},
	)
	if err != nil {
		return err
	}
	for _, bdcItem := range bdcList.Items {
		bdcItem := bdcItem // pin it
		bdcObj := &bdcItem
		bdcObj.Finalizers = util.RemoveString(bdcObj.Finalizers, types.CSPCFinalizer)
		bdcObj, err = c.GetStoredOpenebsVersionClient().BlockDeviceClaims(cspc.Namespace).Update(context.TODO(), bdcObj, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to remove finalizers from bdc %s", bdcItem.Name)
		}
	}

	cspc.RemoveFinalizer(types.CSPCFinalizer)
	_, err = c.GetStoredCStorVersionClient().CStorPoolClusters(cspc.Namespace).Update(context.TODO(), cspc, metav1.UpdateOptions{})

	if err != nil {
		return errors.Wrap(err, "failed to remove CSPC finalizer on cspc resource")
	}
	return nil
}

func (c *Controller) populateDesiredInstances(cspc *cstor.CStorPoolCluster) (*cstor.CStorPoolCluster, error) {
	cspc.Status.DesiredInstances = int32(len(cspc.Spec.Pools))

	cspc, err := c.GetStoredCStorVersionClient().
		CStorPoolClusters(cspc.Namespace).
		Update(context.TODO(), cspc, metav1.UpdateOptions{})

	if err != nil {
		return nil, errors.Wrapf(
			err,
			"failed to update cspc %s while adding desired instances number in spec",
			cspc.Name,
		)
	}
	return cspc, nil
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
			Update(context.TODO(), cspc, metav1.UpdateOptions{})

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
			context.TODO(),
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

func (c *Controller) reconcileVersion(cspc *cstor.CStorPoolCluster) (*cstor.CStorPoolCluster, error) {
	var err error
	// the below code uses deep copy to have the state of object just before
	// any update call is done so that on failure the last state object can be returned
	if cspc.VersionDetails.Status.Current != cspc.VersionDetails.Desired {
		if !version.IsCurrentVersionValid(cspc.VersionDetails.Status.Current) {
			return cspc, errors.Errorf("invalid current version %s", cspc.VersionDetails.Status.Current)
		}
		if !version.IsDesiredVersionValid(cspc.VersionDetails.Desired) {
			return cspc, errors.Errorf("invalid desired version %s", cspc.VersionDetails.Desired)
		}
		cspcObj := cspc.DeepCopy()
		if cspc.VersionDetails.Status.State != cstor.ReconcileInProgress {
			cspcObj.VersionDetails.Status.SetInProgressStatus()
			cspcObj, err = c.clientset.CstorV1().CStorPoolClusters(cspcObj.Namespace).Update(context.TODO(), cspcObj, metav1.UpdateOptions{})
			if err != nil {
				return cspc, err
			}
		}
		// As no other steps are required just change current version to
		// desired version
		path := strings.Split(cspcObj.VersionDetails.Status.Current, "-")[0]
		u := &upgradeParams{
			cspc:   cspcObj,
			client: c.clientset,
		}
		// Get upgrade function for corresponding path, if path does not
		// exits then no upgrade is required and funcValue will be nil.
		funcValue := upgradeMap[path]
		if funcValue != nil {
			cspcObj, err = funcValue(u)
			if err != nil {
				return cspcObj, err
			}
		}
		cspc = cspcObj.DeepCopy()
		cspcObj.VersionDetails.SetSuccessStatus()
		cspcObj, err = c.clientset.CstorV1().CStorPoolClusters(cspcObj.Namespace).Update(context.TODO(), cspcObj, metav1.UpdateOptions{})
		if err != nil {
			return cspc, errors.Wrap(err, "failed to update CSPC")
		}
		return cspcObj, nil
	}
	return cspc, nil
}

// GetCSPIWithoutDeployment gets the CSPIs for whom the pool deployment does not exists.
func (c *Controller) GetCSPIWithoutDeployment(cspc *cstor.CStorPoolCluster) ([]cstor.CStorPoolInstance, error) {
	var cspiList []cstor.CStorPoolInstance
	cspiGotList, err := c.GetStoredCStorVersionClient().CStorPoolInstances(cspc.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: string(types.CStorPoolClusterLabelKey) + "=" + cspc.Name})
	if err != nil {
		return nil, errors.Wrapf(err, "could not list cspi for cspc {%s}", cspc.Name)
	}
	for _, cspObj := range cspiGotList.Items {
		cspObj := cspObj
		_, err := c.kubeclientset.AppsV1().Deployments(cspc.Namespace).Get(context.TODO(), cspObj.Name, metav1.GetOptions{})
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

/*
syncCSPI propagates all the required changes from cspc to respective cspi.
ToDo: Currently -- in every resync interval the sync is tried and this needs to be improved by queuing cspc
only at times when it is required.
*/

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
			klog.Errorf("failed to sync cspi %s from its parent cspc %s: {%s}", cspi.Name, cspc.Name, err)
		}
	}
	return nil
}

// syncCSPIWithCSPC syncs a cspi with the parent cspc and hence updates the cspi's corresponding
// pool manager deployment if required.
func (pc *PoolConfig) syncCSPIWithCSPC(cspc *cstor.CStorPoolCluster, cspi *cstor.CStorPoolInstance) error {
	cspiCopy := cspi.DeepCopy()
	klog.V(2).Infof("Syncing cspi %s from parent cspc %s", cspiCopy.Name, cspc.Name)

	// Finding out the cspc pool spec in the following way:
	// 1. Using node selectors i.e if node selector of cspc and cspi is
	// matched then that is the spec of CSPC belongs to searching CSPI
	//			(If above step failed then find using step2)
	// 2. Find out using data raidgroup blockdevice names i.e if atleast
	// one blockdevice name in CSPI data raid groups is matched to
	// blockdevice name in data raidgroups in CSPC spec then

	//NOTE: CSPC pools spec consist repetion of blockdevice names
	// is outof context. This kind of CSPC should not be reconciled
	for _, poolSpec := range cspc.Spec.Pools {

		if reflect.DeepEqual(poolSpec.NodeSelector, cspi.Spec.NodeSelector) ||
			pc.isCSPISpecExist([]cstor.PoolSpec{poolSpec}, cspi.Spec) {

			poolSpec := poolSpec
			cspiCopy = cspiCopy.WithPoolConfig(poolSpec.PoolConfig).
				WithNodeSelectorByReference(poolSpec.NodeSelector)
			defaultPoolConfig(cspiCopy, cspc)
			hostName, err := pc.AlgorithmConfig.GetNodeFromLabelSelector(cspiCopy.Spec.NodeSelector)
			if err != nil || hostName == "" {
				return errors.Errorf("could not use node for selectors {%v}: {%s}", cspiCopy.Spec.NodeSelector, err.Error())
			}
			cspiCopy.Spec.HostName = hostName
			break
		}
	}
	gotCSPI, err := pc.Controller.GetStoredCStorVersionClient().CStorPoolInstances(cspiCopy.Namespace).Update(context.TODO(), cspiCopy, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return pc.patchPoolDeploymentSpec(gotCSPI)
}

/*
defaultPoolConfig defaults the value of the required fields in the cspi
defaulting mechanism is -- if certain fields are not specified in cspc at
pool config level ( or per cspi level ) then a generic value from pool spec
is taken.
Please refer following design document to understand more
https://github.com/openebs/api/tree/HEAD/design/cstor/v1

ToDo: Offload this defaulting mechanism to mutating webhook server.
*/
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

	if cspi.Spec.PoolConfig.PriorityClassName == nil {
		priorityClassName := cspc.Spec.DefaultPriorityClassName
		cspi.Spec.PoolConfig.PriorityClassName = &priorityClassName
	}

	if cspi.Spec.PoolConfig.ROThresholdLimit == nil {
		cspi.Spec.PoolConfig.ROThresholdLimit = &defaultROThresholdLimit
	}
}

/*
patchPoolDeploymentSpec calculates the diff (let us call this as 2-way patch data) between current existing pool
manager deployment and desired existing pool manager.
(The spec for desired pool manager is calculated/given/ordered by cspc-operator.)
Once the 2-way patch data is calculated -- a strategic merge patch data is obtained by passing the current pool
manager spec and 2-way patch data.
Finally, a strategic merge patch is applied by using the strategic merge patch data.
To understand more on how strategic merge patch works refer following doc:
https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md

ToDo: Explore server side apply -- right now server side apply is still worked upon in k8s community.

NOTE : A strategic merge patch happens only to specific fields in a certain native k8s object.
For example, fields like `tolerations` in a deployment object cannot have strategic merge patch and are always
a JSON Merge Patch ( RFC 6902 )

*/
func (pc *PoolConfig) patchPoolDeploymentSpec(cspi *cstor.CStorPoolInstance) error {
	// Get the corresponding deployment for the cspi

	existingDeployObj, err := pc.Controller.kubeclientset.AppsV1().Deployments(cspi.Namespace).Get(context.TODO(), cspi.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get corresponding pool manager deployment for cspi %s in namespace %s", cspi.Name, cspi.Name)
	}

	newDeployObj := pc.AlgorithmConfig.GetPoolDeploySpec(cspi)

	existingDeployObjInBytes, err := json.Marshal(existingDeployObj)

	if err != nil {
		return errors.Wrapf(err, "failed to marshal existing deployment object %s", existingDeployObj.Name)
	}

	newDeployObjInBytes, err := json.Marshal(newDeployObj)

	if err != nil {
		return errors.Wrapf(err, "failed to marshal new deployment object for existing deployment %s", existingDeployObj.Name)
	}

	twoWayPatchData, err := strategicpatch.CreateTwoWayMergePatch(existingDeployObjInBytes, newDeployObjInBytes, v1.Deployment{}, []mergepatch.PreconditionFunc{}...)
	if err != nil {
		return errors.Wrap(err, "could not compute two way patch data")
	}

	strategicPatchData, err := strategicpatch.StrategicMergePatch(existingDeployObjInBytes, twoWayPatchData, v1.Deployment{})
	if err != nil {
		return errors.Wrap(err, "could not compute strategic patch data")
	}

	_, err = pc.Controller.kubeclientset.AppsV1().Deployments(cspi.Namespace).Patch(context.TODO(), existingDeployObj.Name, k8stypes.StrategicMergePatchType, strategicPatchData, metav1.PatchOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to patch pool manager")
	}

	return nil
}

func (c *Controller) ShouldReconcile(cspc cstor.CStorPoolCluster) (bool, string) {
	cspcOperatorVersion := version.Current()
	cspcVersion := cspc.VersionDetails.Status.Current
	if cspcVersion == "" {
		return true, ""
	}

	if cspcVersion != cspcOperatorVersion {
		return false, fmt.Sprintf("cspc operator version is %s but cspc version is %s", cspcOperatorVersion, cspcVersion)
	}

	return true, ""
}
