/*
Copyright 2020 The OpenEBS Authors.

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

package cspicontroller

import (
	"context"
	"fmt"
	"os"
	"strings"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/openebs/cstor-operators/pkg/controllers/common"
	cspiutil "github.com/openebs/cstor-operators/pkg/controllers/cspi-controller/util"
	zpool "github.com/openebs/cstor-operators/pkg/pool/operations"
	"github.com/openebs/cstor-operators/pkg/version"
	"github.com/openebs/cstor-operators/pkg/volumereplica"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

type upgradeParams struct {
	cspi   *cstor.CStorPoolInstance
	client clientset.Interface
}

type upgradeFunc func(u *upgradeParams) (*cstor.CStorPoolInstance, error)

var (
	upgradeMap = map[string]upgradeFunc{}
)

// reconcile will ensure that pool for given
// key is created and running
func (c *CStorPoolInstanceController) reconcile(key string) error {
	var err error
	var isImported bool

	cspi, err := c.getCSPIObjFromKey(key)
	if err != nil || cspi == nil {
		return err
	}

	if IsReconcileDisabled(cspi) {
		c.recorder.Event(cspi,
			corev1.EventTypeWarning,
			fmt.Sprintf("reconcile is disabled via %q annotation", types.OpenEBSDisableReconcileLabelKey),
			"Skipping reconcile")
		return nil
	}

	if cspi.IsDestroyed() {
		return c.destroy(cspi)
	}

	cspiObj, err := c.addPoolProtectionFinalizer(cspi)
	if err != nil {
		c.recorder.Event(cspi,
			corev1.EventTypeWarning,
			fmt.Sprintf("Failed to add %s finalizer.", types.PoolProtectionFinalizer),
			err.Error())
		return nil
	}

	cspiObj, err = c.reconcileVersion(cspiObj)
	if err != nil {
		message := fmt.Sprintf("Failed to upgrade cspi to %s version: %s",
			cspiObj.VersionDetails.Desired,
			err.Error())
		klog.Errorf("failed to upgrade cspi %s:%s", cspiObj.Name, err.Error())
		c.recorder.Event(cspiObj, corev1.EventTypeWarning, "FailedUpgrade", message)
		cspiObj.VersionDetails.Status.SetErrorStatus(
			"Failed to reconcile cspi version",
			err,
		)
		_, err = c.clientset.CstorV1().CStorPoolInstances(cspiObj.Namespace).Update(context.TODO(), cspiObj, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update versionDetails status for cspi %s:%s", cspiObj.Name, err.Error())
		}
		return nil
	}

	cspi = cspiObj

	// validate CSPI validates the CSPI
	err = validateCSPI(cspi)
	if err != nil {
		c.recorder.Event(cspi,
			corev1.EventTypeWarning,
			"Validation failed",
			err.Error())
		return nil
	}

	// Instantiate the pool operation config
	// ToDo: NewOperationsConfig is used a other handlers e.g. destroy: fix the repeatability.
	oc := zpool.NewOperationsConfig().
		WithKubeClientSet(c.kubeclientset).
		WithOpenEBSClient(c.clientset).
		WithRecorder(c.recorder).
		WithZcmdExecutor(c.zcmdExecutor)

	// take a lock for common package for updating variables
	common.SyncResources.Mux.Lock()

	// try to import pool
	isImported, err = oc.Import(cspi)
	if isImported {
		if err != nil {
			common.SyncResources.Mux.Unlock()
			c.recorder.Event(cspi,
				corev1.EventTypeWarning,
				string(common.FailureImported),
				fmt.Sprintf("Failed to import pool due to '%s'", err.Error()))
			return nil
		}
		zpool.CheckImportedPoolVolume()
		common.SyncResources.Mux.Unlock()
		cspiGot, err := c.update(cspi)
		if err != nil {
			c.recorder.Event(cspiGot,
				corev1.EventTypeWarning,
				string(common.FailedSynced),
				err.Error())
		}

		// If everything is alright here -- sync the cspi
		// Note: Even if update fails, cspiGot will not be nil.
		// In case of failed update, passed cspi to update functions
		// is returned.
		c.sync(cspiGot)

		return nil
	}

	if cspi.IsEmptyStatus() || cspi.IsPendingStatus() {
		err = oc.Create(cspi)
		if err != nil {
			// We will try to create it in next event
			c.recorder.Event(cspi,
				corev1.EventTypeWarning,
				string(common.FailureCreate),
				fmt.Sprintf("Failed to create pool due to '%s'", err.Error()))

			_ = oc.Delete(cspi)
			common.SyncResources.Mux.Unlock()
			return nil
		}
		common.SyncResources.Mux.Unlock()

		c.recorder.Event(cspi,
			corev1.EventTypeNormal,
			string(common.SuccessCreated),
			string("Pool created successfully"))

		cspiGot, err := c.update(cspi)
		if err != nil {
			c.recorder.Event(cspiGot,
				corev1.EventTypeWarning,
				string(common.FailedSynced),
				err.Error())
		}
		return nil

	}
	common.SyncResources.Mux.Unlock()
	// This case is possible incase of ephemeral disks
	if !cspi.IsEmptyStatus() && !cspi.IsPendingStatus() {
		// This scenario will occur when the zpool command hung due to same/someother
		// bad disk existence in the system (or) when the underlying pool disk is lost.

		// NOTE: If zpool command hung cstor-pool container in pool manager will restart
		// because of due liveness on the pool. If cstor-pool container is killed then
		// zpool commands will error out and fall into this scenario
		c.recorder.Event(cspi, corev1.EventTypeWarning,
			string(common.FailedSynced),
			string("Failed to import the pool as the underlying pool might be lost or the disk(s) has gone bad"),
		)
		// Set Pool Lost condition to true
		condition := cspiutil.NewCSPICondition(
			cstor.CSPIPoolLost,
			corev1.ConditionTrue,
			"PoolLost", "failed to import"+zpool.PoolName()+"pool")
		cspi, _ = c.UpdateStatusConditionEventually(cspi, *condition)
	}
	return nil
}

func (c *CStorPoolInstanceController) destroy(cspi *cstor.CStorPoolInstance) error {
	var phase cstor.CStorPoolInstancePhase

	if !util.ContainsString(cspi.Finalizers, types.PoolProtectionFinalizer) {
		return nil
	}
	// Instantiate the pool operation config
	oc := zpool.NewOperationsConfig().
		WithKubeClientSet(c.kubeclientset).
		WithOpenEBSClient(c.clientset).
		WithZcmdExecutor(c.zcmdExecutor)

	// DeletePool is to delete cstor zpool.
	// It will also clear the label for relevant disk
	err := oc.Delete(cspi)
	if err != nil {
		c.recorder.Event(cspi,
			corev1.EventTypeWarning,
			string(common.FailureDestroy),
			fmt.Sprintf("Failed to delete pool due to '%s'", err.Error()))
		phase = cstor.CStorPoolStatusDeletionFailed
		goto updatestatus
	}

	// removeFinalizer is to remove finalizer of cStorPoolInstance resource.
	err = c.removeFinalizer(cspi)
	if err != nil {
		// Object will exist. Let's set status as offline
		klog.Errorf("removeFinalizer failed %s", err.Error())
		phase = cstor.CStorPoolStatusDeletionFailed
		goto updatestatus
	}
	klog.Infof("Pool %s deleted successfully", cspi.Name)
	return nil

updatestatus:
	cspi.Status.Phase = phase
	if _, er := c.clientset.
		CstorV1().
		CStorPoolInstances(cspi.Namespace).
		Update(context.TODO(), cspi, metav1.UpdateOptions{}); er != nil {
		klog.Errorf("Update failed %s", er.Error())
	}
	return err
}

func (c *CStorPoolInstanceController) update(cspi *cstor.CStorPoolInstance) (*cstor.CStorPoolInstance, error) {
	oc := zpool.NewOperationsConfig().
		WithKubeClientSet(c.kubeclientset).
		WithOpenEBSClient(c.clientset).
		WithRecorder(c.recorder).
		WithZcmdExecutor(c.zcmdExecutor)
	ncspi, err := oc.Update(cspi)
	if err != nil {
		return ncspi, errors.Errorf("Failed to update pool due to %s", err.Error())
	}
	return c.updateStatus(ncspi)
}

func (c *CStorPoolInstanceController) updateStatus(cspi *cstor.CStorPoolInstance) (*cstor.CStorPoolInstance, error) {
	// ToDo: Use the status from the cspi object that is passed in arg else other fields
	// might get lost.
	var status cstor.CStorPoolInstanceStatus
	pool := zpool.PoolName()
	propertyList := []string{"health", "io.openebs:readonly"}
	oc := zpool.NewOperationsConfig().
		WithZcmdExecutor(c.zcmdExecutor)

	// Since we queried in following order health and io.openebs:readonly output also
	// will be in same order
	valueList, err := oc.GetListOfPropertyValues(pool, propertyList)
	if err != nil {
		return cspi, errors.Errorf("Failed to fetch %v output: %v error: %v", propertyList, valueList, err)
	} else {
		// valueList[0] will hold the value of health of cStor pool
		// valueList[1] will hold the value of io.openebs:readonly of cStor pool
		status.Phase = cstor.CStorPoolInstancePhase(valueList[0])
		if valueList[1] == "on" {
			status.ReadOnly = true
		}
	}

	provisionedRepCount, healthyRepCount, err := volumereplica.GetProvisionedAndHealthyReplicaCount(c.zcmdExecutor)
	if err != nil {
		klog.Errorf("failed to get provisioned and healthy replica count %s", err.Error())
	} else {
		status.ProvisionedReplicas = provisionedRepCount
		status.HealthyReplicas = healthyRepCount
	}

	status.Capacity, err = oc.GetCSPICapacity(pool)
	if err != nil {
		return cspi, errors.Errorf("Failed to sync due to %s", err.Error())
	}
	c.updateROMode(&status, *cspi)
	// addDiskUnavailableCondition will add DiskUnavailable condition on cspi status
	c.addDiskUnavailableCondition(cspi)
	// Point to existing conditions
	status.Conditions = cspi.Status.Conditions

	if IsStatusChange(cspi.Status, status) {
		cspi.Status = status
		cspiGot, err := c.clientset.
			CstorV1().
			CStorPoolInstances(cspi.Namespace).
			Update(context.TODO(), cspi, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Error %v", err)
			return cspi, errors.Errorf("Failed to updateStatus due to '%s'", err.Error())
		}
		return cspiGot, nil
	}

	return cspi, nil
}

// updateROMode sets/unsets the pool readonly mode property. It does the following changes
// 1. If pool used space reached to roThresholdLimit then pool will be set to readonly mode
// 2. If pool was in readonly mode if roThresholdLimit/pool expansion was happened then it
//    unsets the ReadOnly Mode.
// NOTE: This function must be invoked after having the updated
//       cspiStatus information from zfs/zpool
func (c *CStorPoolInstanceController) updateROMode(
	cspiStatus *cstor.CStorPoolInstanceStatus, cspi cstor.CStorPoolInstance) {
	roThresholdLimit := 85

	if cspi.Spec.PoolConfig.ROThresholdLimit != nil {
		roThresholdLimit = *cspi.Spec.PoolConfig.ROThresholdLimit
	}
	availableInBytes := cspiStatus.Capacity.Free.Value()
	usedInBytes := cspiStatus.Capacity.Used.Value()
	totalInBytes := availableInBytes + usedInBytes
	pool := zpool.PoolName()
	oc := zpool.NewOperationsConfig().
		WithZcmdExecutor(c.zcmdExecutor)

	usedPercentage := (usedInBytes * 100) / totalInBytes
	// If roThresholdLimit sets 100% and pool used storage reached to 100%
	// then there might be chances that operations will hung so it is not
	// recommended to perform operations
	if (int(usedPercentage) >= roThresholdLimit) && roThresholdLimit != 100 {
		if !cspiStatus.ReadOnly {
			if err := oc.SetPoolRDMode(pool, true); err != nil {
				// Here, we are just logging in next reconciliation it will be retried
				klog.Errorf("failed to set pool ReadOnly Mode to %t error: %s", true, err.Error())
			} else {
				cspiStatus.ReadOnly = true
				c.recorder.Event(&cspi,
					corev1.EventTypeWarning,
					"PoolReadOnlyThreshold",
					"Pool storage limit reached to read only threshold limit. "+
						"Pool expansion is required to make its volume replicas RW",
				)
			}
		}
	} else {
		if cspiStatus.ReadOnly {
			if err := oc.SetPoolRDMode(pool, false); err != nil {
				klog.Errorf("Failed to unset pool readOnly mode : %v", err)
			} else {
				cspiStatus.ReadOnly = false
				c.recorder.Event(&cspi,
					corev1.EventTypeNormal,
					"PoolReadOnlyThreshold",
					"Pool roThresholdLimit or pool got expanded due to that pool readOnly mode is unset",
				)
			}

		}
	}
}

// getCSPIObjFromKey returns object corresponding to the resource key
func (c *CStorPoolInstanceController) getCSPIObjFromKey(key string) (*cstor.CStorPoolInstance, error) {
	// Convert the key(namespace/name) string into a distinct name and namespace
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil, nil
	}

	cspi, err := c.clientset.
		CstorV1().
		CStorPoolInstances(ns).
		Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		// The cStorPoolInstance resource may no longer exist, in which case we stop
		// processing.
		if k8serror.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("CSPI '%s' in work queue no longer exists", key))
			return nil, nil
		}

		return nil, err
	}
	return cspi, nil
}

// removeFinalizer is to remove finalizer of cstorpoolinstance resource.
func (c *CStorPoolInstanceController) removeFinalizer(cspi *cstor.CStorPoolInstance) error {
	if len(cspi.Finalizers) == 0 {
		return nil
	}
	cspi.Finalizers = util.RemoveString(cspi.Finalizers, types.PoolProtectionFinalizer)
	_, err := c.clientset.
		CstorV1().
		CStorPoolInstances(cspi.Namespace).
		Update(context.TODO(), cspi, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	klog.Infof("Removed Finalizer: %v, %v",
		cspi.Name,
		string(cspi.GetUID()))
	return nil
}

// addPoolProtectionFinalizer is to add PoolProtectionFinalizer finalizer of cstorpoolinstance resource.
func (c *CStorPoolInstanceController) addPoolProtectionFinalizer(
	cspi *cstor.CStorPoolInstance) (*cstor.CStorPoolInstance, error) {
	// if PoolProtectionFinalizer is already present return
	if util.ContainsString(cspi.Finalizers, types.PoolProtectionFinalizer) {
		return cspi, nil
	}
	cspi.Finalizers = append(cspi.Finalizers, types.PoolProtectionFinalizer)
	newCSPI, err := c.clientset.
		CstorV1().
		CStorPoolInstances(cspi.Namespace).
		Update(context.TODO(), cspi, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	klog.Infof("Added Finalizer: %v, %v",
		cspi.Name,
		string(cspi.GetUID()))
	return newCSPI, nil
}

func (c *CStorPoolInstanceController) sync(cspi *cstor.CStorPoolInstance) {
	oc := zpool.NewOperationsConfig().
		WithZcmdExecutor(c.zcmdExecutor)

	// reconcile pool(fs & pool both) properties
	err := oc.SetPoolProperties(cspi)
	if err != nil {
		c.recorder.Event(cspi,
			corev1.EventTypeWarning,
			"Pool "+string("FailedToSetPoolProperties"),
			fmt.Sprintf("Failed to set pool properties: %v", err.Error()))
	}
}

func (c *CStorPoolInstanceController) addDiskUnavailableCondition(cspi *cstor.CStorPoolInstance) {
	diskUnavailableCondition := cspiutil.GetCSPICondition(cspi.Status, cstor.CSPIDiskUnavailable)
	oc := zpool.NewOperationsConfig().
		WithKubeClientSet(c.kubeclientset).
		WithOpenEBSClient(c.clientset).
		WithRecorder(c.recorder).
		WithZcmdExecutor(c.zcmdExecutor)
	unAvailableDisks, err := oc.GetUnavailableDiskList(cspi)

	if err != nil {
		klog.Errorf("failed to get unavailable disks error: %v", err)
		return
	}
	if len(unAvailableDisks) > 0 {
		newCondition := cspiutil.NewCSPICondition(
			cstor.CSPIDiskUnavailable,
			corev1.ConditionTrue,
			"DisksAreUnavailable",
			fmt.Sprintf("Following disks %v are unavailable/faulted", unAvailableDisks))
		cspiutil.SetCSPICondition(&cspi.Status, *newCondition)
	} else {
		if diskUnavailableCondition != nil {
			newCondition := cspiutil.NewCSPICondition(
				cstor.CSPIDiskUnavailable,
				corev1.ConditionFalse,
				"DisksAreAvailable",
				"")
			cspiutil.SetCSPICondition(&cspi.Status, *newCondition)
		}
	}
}

func (c *CStorPoolInstanceController) reconcileVersion(cspi *cstor.CStorPoolInstance) (*cstor.CStorPoolInstance, error) {
	var err error
	// the below code uses deep copy to have the state of object just before
	// any update call is done so that on failure the last state object can be returned
	if cspi.VersionDetails.Status.Current != cspi.VersionDetails.Desired {
		if !version.IsCurrentVersionValid(cspi.VersionDetails.Status.Current) {
			return cspi, errors.Errorf("invalid current version %s", cspi.VersionDetails.Status.Current)
		}
		if !version.IsDesiredVersionValid(cspi.VersionDetails.Desired) {
			return cspi, errors.Errorf("invalid desired version %s", cspi.VersionDetails.Desired)
		}
		cspiObj := cspi.DeepCopy()
		if cspi.VersionDetails.Status.State != cstor.ReconcileInProgress {
			cspiObj.VersionDetails.Status.SetInProgressStatus()
			cspiObj, err = c.clientset.CstorV1().CStorPoolInstances(cspiObj.Namespace).Update(context.TODO(), cspiObj, metav1.UpdateOptions{})
			if err != nil {
				return cspi, err
			}
		}
		// As no other steps are required just change current version to
		// desired version
		path := strings.Split(cspiObj.VersionDetails.Status.Current, "-")[0]
		u := &upgradeParams{
			cspi:   cspiObj,
			client: c.clientset,
		}
		// Get upgrade function for corresponding path, if path does not
		// exits then no upgrade is required and funcValue will be nil.
		funcValue := upgradeMap[path]
		if funcValue != nil {
			cspiObj, err = funcValue(u)
			if err != nil {
				return cspiObj, err
			}
		}
		cspi = cspiObj.DeepCopy()
		cspiObj.VersionDetails.SetSuccessStatus()
		cspiObj, err = c.clientset.CstorV1().CStorPoolInstances(cspiObj.Namespace).Update(context.TODO(), cspiObj, metav1.UpdateOptions{})
		if err != nil {
			return cspi, errors.Wrap(err, "failed to update CSPI")
		}
		return cspiObj, nil
	}
	return cspi, nil
}

// markCSPIStatusToOffline will fetch all the CSPI resources present
// in etcd and mark it's own CSPI.Status to Offline
func (c *CStorPoolInstanceController) markCSPIStatusToOffline() {
	cspiList, err := c.clientset.CstorV1().CStorPoolInstances("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf("Failed to fetch CSPI list error: %v", err)
	}
	// Fetch CSPI UUID of this pool-manager
	cspiID := os.Getenv(OpenEBSIOCSPIID)
	for _, cspi := range cspiList.Items {
		if string(cspi.GetUID()) == cspiID {
			// If pool-manager container restarts or pod is deleted
			// before even creating a pool then updating status to
			// Offline will never create a pool in subsequent reconciliations
			// until CSPI Phase is updated to "" or pending
			if cspi.Status.Phase == "" || cspi.Status.Phase == cstor.CStorPoolStatusPending {
				return
			}
			cspi.Status.Phase = cstor.CStorPoolStatusOffline
			// There will be one-to-one mapping between CSPI and pool-manager
			// So after finding good to break
			_, err = c.clientset.CstorV1().CStorPoolInstances(cspi.Namespace).Update(context.TODO(), &cspi, metav1.UpdateOptions{})
			if err != nil {
				klog.Errorf("Failed to update CSPI: %s status to %s", cspi.Name, cstor.CStorPoolStatusOffline)
				return
			}
			klog.Infof("Status marked %s for CSPI: %s", cstor.CStorPoolStatusOffline, cspi.Name)
			break
		}
	}
}

// validateCSPI returns error if CSPI spec validation fails otherwise nil
func validateCSPI(cspi *cstor.CStorPoolInstance) error {
	if len(cspi.Spec.DataRaidGroups) == 0 {
		return errors.Errorf("No data RaidGroups exists")
	}
	if cspi.Spec.PoolConfig.DataRaidGroupType == "" {
		return errors.Errorf("Missing DataRaidGroupType")
	}
	if len(cspi.Spec.WriteCacheRaidGroups) != 0 &&
		cspi.Spec.PoolConfig.WriteCacheGroupType == "" {
		return errors.Errorf("Missing WriteCacheRaidGroupType")
	}
	for _, rg := range cspi.Spec.DataRaidGroups {
		if len(rg.CStorPoolInstanceBlockDevices) == 0 {
			return errors.Errorf("No BlockDevices exist in one of the DataRaidGroup")
		}
	}
	for _, rg := range cspi.Spec.WriteCacheRaidGroups {
		if len(rg.CStorPoolInstanceBlockDevices) == 0 {
			return errors.Errorf("No BlockDevices exist in one of the WriteCache RaidGroup")
		}
	}
	return nil
}
