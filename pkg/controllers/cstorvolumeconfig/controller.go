/*
Copyright 2019 The OpenEBS Authors

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

package cstorvolumeconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	apitypes "github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/openebs/cstor-operators/pkg/util/hash"
	"github.com/openebs/cstor-operators/pkg/version"
	errors "github.com/pkg/errors"
	"k8s.io/klog"

	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	ref "k8s.io/client-go/tools/reference"
)

const (
	// SuccessSynced is used as part of the Event 'reason' when a
	// cstorvolumeconfig is synced
	SuccessSynced = "Synced"
	// Provisioning is used as part of the Event 'reason' when a
	// cstorvolumeconfig is in provisioning stage
	Provisioning = "Provisioning"
	// ErrResourceExists is used as part of the Event 'reason' when a
	// cstorvolumeconfig fails to sync due to a cstorvolumeconfig of the same
	// name already existing.
	ErrResourceExists = "ErrResourceExists"
	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a cstorvolumeconfig already existing
	MessageResourceExists = "Resource %q already exists and is not managed by CVC"
	// MessageResourceSynced is the message used for an Event fired when a
	// cstorvolumeconfig is synced successfully
	MessageResourceSynced = "cstorvolumeconfig synced successfully"
	// MessageResourceCreated msg used for cstor volume provisioning success event
	MessageResourceCreated = "cstorvolumeconfig created successfully"
	// MessageCVCPublished msg used for cstor volume provisioning publish events
	MessageCVCPublished = "cstorvolumeconfig %q must be published/attached on node"
	// CStorVolumeConfigFinalizer name of finalizer on CStorVolumeConfig that
	// are bound by CStorVolume
	CStorVolumeConfigFinalizer = "cvc.openebs.io/finalizer"
	// DeProvisioning is used as part of the event 'reason' during
	// cstorvolumeconfig deprovisioning stage
	DeProvisioning = "DeProvisioning"
)

var knownResizeConditions = map[apis.CStorVolumeConfigConditionType]bool{
	apis.CStorVolumeConfigResizing:      true,
	apis.CStorVolumeConfigResizePending: true,
}

// Patch struct represent the struct used to patch
// the cstorvolumeconfig object
type Patch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

type upgradeParams struct {
	cvc    *apis.CStorVolumeConfig
	client clientset.Interface
}

type upgradeFunc func(u *upgradeParams) (*apis.CStorVolumeConfig, error)

var (
	upgradeMap = map[string]upgradeFunc{}
)

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the spcPoolUpdated resource
// with the current status of the resource.
func (c *CVCController) syncHandler(key string) error {
	startTime := time.Now()
	klog.V(4).Infof("Started syncing cstorvolumeconfig %q (%v)", key, startTime)
	defer func() {
		klog.V(4).Infof("Finished syncing cstorvolumeconfig %q (%v)", key, time.Since(startTime))
	}()

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the cvc resource with this namespace/name
	cvc, err := c.cvcLister.CStorVolumeConfigs(namespace).Get(name)
	if k8serror.IsNotFound(err) {
		runtime.HandleError(fmt.Errorf("cstorvolumeconfig '%s' has been deleted", key))
		return nil
	}
	if err != nil {
		return err
	}
	cvcCopy := cvc.DeepCopy()
	err = c.syncCVC(cvcCopy)
	return err
}

// enqueueCVC takes a CVC resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than CStorVolumeConfigs.
func (c *CVCController) enqueueCVC(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.Add(key)

	/*	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
			obj = unknown.Obj
		}
		if cvc, ok := obj.(*apis.CStorVolumeConfig); ok {
			objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(cvc)
			if err != nil {
				klog.Errorf("failed to get key from object: %v, %v", err, cvc)
				return
			}
			klog.V(5).Infof("enqueued %q for sync", objName)
			c.workqueue.Add(objName)
		}
	*/
}

// synCVC is the function which tries to converge to a desired state for the
// CStorVolumeConfigs
func (c *CVCController) syncCVC(cvc *apis.CStorVolumeConfig) error {

	var err error

	updatedCVC, err := c.populateVersion(cvc)
	if err != nil {
		klog.Errorf("failed to add versionDetails to CVC %s in namespace %s :{%s}", cvc.Name, cvc.Namespace, err.Error())
		return nil
	}

	cvc, err = c.reconcileVersion(updatedCVC)
	if err != nil {
		message := fmt.Sprintf("Failed to upgrade cvc to %s version: %s",
			cvc.VersionDetails.Desired,
			err.Error())
		klog.Errorf("failed to upgrade cvc %s:%s", cvc.Name, err.Error())
		c.recorder.Event(cvc, corev1.EventTypeWarning, "FailedUpgrade", message)
		cvc.VersionDetails.Status.SetErrorStatus(
			"Failed to reconcile cvc version",
			err,
		)
		_, err = c.clientset.CstorV1().CStorVolumeConfigs(cvc.Namespace).Update(context.TODO(), cvc, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update versionDetails status for cvc %s:%s", cvc.Name, err.Error())
		}
		return nil
	}

	// CStor Volume Claim should be deleted. Check if deletion timestamp is set
	// and remove finalizer.
	if c.isClaimDeletionCandidate(cvc) {
		klog.Infof("syncClaim: remove finalizer for CStorVolumeConfigVolume [%s]", cvc.Name)
		err = c.removeClaimFinalizer(cvc)
		if err != nil {
			c.recorder.Eventf(cvc, corev1.EventTypeWarning, DeProvisioning, err.Error())
		}
		return nil
	}

	if ok, reason := c.ShouldReconcile(cvc); !ok {
		// Do not reconcile cvc if version mismatched
		message := fmt.Sprintf("can not reconcile CVC %s as %s", cvc.Name, reason)
		c.recorder.Event(cvc, corev1.EventTypeWarning, "CVC Reconcile", message)
		klog.Warningf("Cannot not reconcile CVC %s in namespace %s as %s", cvc.Name, cvc.Namespace, reason)
		return nil
	}

	volName := cvc.Name
	if volName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("%+v: cvc name must be specified", cvc))
		return nil
	}

	if cvc.Status.Phase == apis.CStorVolumeConfigPhasePending {
		klog.V(2).Infof("provisioning cstor volume %+v", cvc)
		_, err = c.createVolumeOperation(cvc)
		if err != nil {
			//Record an event to indicate that any provisioning operation is failed.
			c.recorder.Eventf(cvc, corev1.EventTypeWarning, Provisioning, err.Error())
		}
	}
	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	if c.cvcNeedResize(cvc) {
		err = c.resizeCVC(cvc)
	}
	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	if c.isCVCScalePending(cvc) {
		// process scale-up/scale-down of volume replicas only if there is
		// change in curent and desired state of replicas pool information
		_ = c.scaleVolumeReplicas(cvc)
	}

	// sync policy changes from cvc.spec.policy e.g. tunables like toleration, resource requirements etc
	return c.syncPolicySpec(cvc)
}

// UpdateCVCObj updates the cstorvolumeconfig object resource to reflect the
// current state of the world
func (c *CVCController) updateCVCObj(
	cvc *apis.CStorVolumeConfig,
	cv *apis.CStorVolume,
) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	cvcCopy := cvc.DeepCopy()
	if cvc.Name != cv.Name {
		return fmt.
			Errorf("could not bind cstorvolumeconfig %s and cstorvolume %s, name does not match",
				cvc.Name,
				cv.Name)
	}

	_, err := c.clientset.CstorV1().CStorVolumeConfigs(cvc.Namespace).Update(context.TODO(), cvcCopy, metav1.UpdateOptions{})

	if err == nil {
		c.recorder.Event(cvc, corev1.EventTypeNormal,
			SuccessSynced,
			MessageResourceCreated,
		)
	}
	return err
}

// createVolumeOperation trigers the all required resource create operation.
// 1. Create volume service.
// 2. Create cstorvolume resource with required iscsi information.
// 3. Create target deployment.
// 4. Create cstorvolumeconfig resource.
// 5. Create PDB provisioning volume is HA volume.
// 6. Update the cstorvolumeconfig with claimRef info, PDB label(only for HA
//    volumes) and bound with cstorvolume.
func (c *CVCController) createVolumeOperation(cvc *apis.CStorVolumeConfig) (*apis.CStorVolumeConfig, error) {

	policyName := cvc.Annotations[string(apitypes.VolumePolicyKey)]
	volumePolicy, err := c.getVolumePolicy(policyName, cvc)
	if err != nil {
		return nil, err
	}

	klog.V(2).Infof("creating cstorvolume service resource")
	svcObj, err := c.getOrCreateTargetService(cvc)
	if err != nil {
		return nil, err
	}

	klog.V(2).Infof("creating cstorvolume resource")
	cvObj, err := c.getOrCreateCStorVolumeResource(svcObj, cvc)
	if err != nil {
		return nil, err
	}

	klog.V(2).Infof("creating cstorvolume target deployment")
	_, err = c.getOrCreateCStorTargetDeployment(cvObj, &volumePolicy.Spec)
	if err != nil {
		return nil, err
	}

	klog.V(2).Infof("creating cstorvolume replica resource")
	err = c.distributePendingCVRs(cvc, cvObj, svcObj, volumePolicy)
	if err != nil {
		return nil, err
	}

	// Fetch the volume replica pool names and use them in PDB and updating in
	// spec and status of CVC
	poolNames, err := GetVolumeReplicaPoolNames(c.clientset, cvc.Name, openebsNamespace)
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to get volume replica pool names of volume %s", cvObj.Name)
	}

	if isHAVolume(cvc) {
		// TODO: When multiple threads or multiple CVC controllers are set then
		// we have to revist entier PDB code path
		var pdbObj *policy.PodDisruptionBudget
		pdbObj, err = c.getOrCreatePodDisruptionBudget(getCSPC(cvc), poolNames)
		if err != nil {
			return nil, errors.Wrapf(err,
				"failed to create PDB for volume: %s", cvc.Name)
		}
		addPDBLabelOnCVC(cvc, pdbObj)
	}

	volumeRef, err := ref.GetReference(scheme.Scheme, cvObj)
	if err != nil {
		return nil, err
	}

	// update the cstorvolume reference, phase as "Bound" and desired
	// capacity
	cvc.Spec.CStorVolumeRef = volumeRef
	cvc.Spec.Policy = volumePolicy.Spec
	cvc.Status.Phase = apis.CStorVolumeConfigPhaseBound
	cvc.Status.Capacity = cvc.Spec.Capacity

	// TODO: Below function needs to be converted into
	// cvc.addReplicaPoolInfo(poolNames) while moving to cstor-operators
	// repo(Currently in Maya writing functions in API package is not encouraged)

	// update volume replica pool information on cvc spec and status
	addReplicaPoolInfo(cvc, poolNames)
	// add hash label in cvc generated from volume policy spec
	addPolicySpecHash(cvc)

	err = c.updateCVCObj(cvc, cvObj)
	if err != nil {
		return nil, err
	}
	return cvc, nil
}

// syncPolicySpec reconcile the policy changes to volume target deployment
// for each volumes based on desired changes under cvc.Spec.Policy
func (c *CVCController) syncPolicySpec(cvc *apis.CStorVolumeConfig) error {
	cvcCopy := cvc.DeepCopy()

	// compare hash label value to the generated hash out of policy changes
	if cvcCopy.Labels[hash.TemplateHashLabelName] != hash.HashObject(cvcCopy.Spec.Policy) {
		klog.V(4).Infof("Initiated policy reconcile for cvc %q :", cvc.Name)
		err := c.patchTargetDeploymentSpec(cvcCopy)
		if err != nil {
			c.recorder.Event(cvcCopy, corev1.EventTypeWarning,
				string("PolicySync"),
				fmt.Sprintf("failed to patch target deployment for cvc %s, err %s ", cvcCopy.Name, err.Error()),
			)
			return err
		}
		// update the hash value in cvc labels generated for new policy changes
		addPolicySpecHash(cvcCopy)
		_, err = c.clientset.CstorV1().CStorVolumeConfigs(cvc.Namespace).Update(context.TODO(), cvcCopy, metav1.UpdateOptions{})
		if err != nil {
			c.recorder.Event(cvcCopy, corev1.EventTypeWarning,
				string("PolicySync"),
				fmt.Sprintf("failed to update hash label in cvc %q, err %s", cvcCopy.Name, err.Error()),
			)
			return err
		}
		c.recorder.Event(cvcCopy, corev1.EventTypeNormal,
			string("PolicySync"),
			fmt.Sprintf("successfully sync policy for cvc %s", cvcCopy.Name),
		)
	}
	return nil
}

func (c *CVCController) patchTargetDeploymentSpec(cvc *apis.CStorVolumeConfig) error {
	orignalDeployObj, err := c.kubeclientset.AppsV1().Deployments(cvc.Namespace).Get(context.TODO(), cvc.Name+"-target", metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get target deployment for volume %s in namespace %s", cvc.Name, cvc.Namespace)
	}

	klog.V(4).Infof("Syncing cvc policy spec \n: %+v", cvc)

	vol, err := c.clientset.CstorV1().CStorVolumes(cvc.Namespace).Get(context.TODO(), cvc.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get cstorvolume {%v}", cvc.Name)
	}

	newDeployObj, err := c.BuildTargetDeployment(vol, &cvc.Spec.Policy)
	if err != nil {
		return errors.Wrapf(err, "failed to build target deployment {%v}", vol.Name)
	}

	oldData, err := json.Marshal(orignalDeployObj)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal original deployment %s", orignalDeployObj.Name)
	}

	newData, err := json.Marshal(newDeployObj)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal updated deployment %s", newDeployObj.Name)
	}

	// CreateTwoWayMergePatch creates a patch that can be passed to StrategicMergePatch from an original
	// document and a modified document, which are passed to the method as json encoded content. It will
	// return a patch that yields the modified document when applied to the original document, or an error
	// if either of the two documents is invalid.
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, appsv1.Deployment{})
	if err != nil {
		return errors.Wrap(err, "failed to create strategic merge patch data")
	}

	_, err = c.kubeclientset.AppsV1().Deployments(cvc.Namespace).Patch(context.TODO(), orignalDeployObj.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to patch volume target deployment")
	}

	return nil
}

func (c *CVCController) getVolumePolicy(
	policyName string,
	cvc *apis.CStorVolumeConfig,
) (*apis.CStorVolumePolicy, error) {

	var err error
	volumePolicy := &apis.CStorVolumePolicy{}

	// Get the default policy
	policySpec := getDefaultPolicySpec()

	if policyName != "" {
		klog.Infof("uses cstorvolume policy %q to configure volume %q", policyName, cvc.Name)
		volumePolicy, err = c.clientset.CstorV1().CStorVolumePolicies(openebsNamespace).Get(context.TODO(), policyName, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"failed to get volume policy %q of volume %q",
				policyName,
				cvc.Name,
			)
		}
		validatePolicySpec(&volumePolicy.Spec)
		return volumePolicy, nil
	}
	// retrun the default policy
	volumePolicy.Spec = policySpec
	return volumePolicy, nil
}

func addPolicySpecHash(cvc *apis.CStorVolumeConfig) {
	labels := hash.SetTemplateHashLabel(cvc.Labels, cvc.Spec.Policy)
	cvc.WithLabels(labels)
}

// isReplicaAffinityEnabled checks if replicaAffinity has been enabled using
// cstor volume policy
func (c *CVCController) isReplicaAffinityEnabled(policy *apis.CStorVolumePolicy) bool {
	return policy.Spec.Provision.ReplicaAffinity
}

// distributePendingCVRs trigers create and distribute pending cstorvolumereplica
// resource among the available cstor pools. This func returns error even when
// required no.of CVRs are Not created
func (c *CVCController) distributePendingCVRs(
	cvc *apis.CStorVolumeConfig,
	cv *apis.CStorVolume,
	service *corev1.Service,
	policy *apis.CStorVolumePolicy,
) error {

	pendingReplicaCount, err := c.getPendingCVRCount(cvc)
	if err != nil {
		return err
	}
	return c.distributeCVRs(pendingReplicaCount, cvc, service, cv, policy)
}

// isClaimDeletionCandidate checks if a cstorvolumeconfig is a deletion candidate.
func (c *CVCController) isClaimDeletionCandidate(cvc *apis.CStorVolumeConfig) bool {
	return cvc.ObjectMeta.DeletionTimestamp != nil &&
		util.ContainsString(cvc.ObjectMeta.Finalizers, CStorVolumeConfigFinalizer)
}

// removeFinalizer removes finalizers present in CStorVolumeConfig resource
// TODO Avoid removing clone finalizer
func (c *CVCController) removeClaimFinalizer(
	cvc *apis.CStorVolumeConfig,
) error {
	if isHAVolume(cvc) {
		err := c.deletePDBIfNotInUse(cvc)
		if err != nil {
			return errors.Wrapf(err,
				"failed to verify whether PDB %s is in use by other volumes",
				getPDBName(cvc),
			)
		}
	}
	cvcPatch := []Patch{
		{
			Op:   "remove",
			Path: "/metadata/finalizers",
		},
	}

	cvcPatchBytes, err := json.Marshal(cvcPatch)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to remove finalizers from cstorvolumeconfig {%s}",
			cvc.Name,
		)
	}

	_, err = c.clientset.
		CstorV1().
		CStorVolumeConfigs(cvc.Namespace).
		Patch(context.TODO(), cvc.Name, types.JSONPatchType, cvcPatchBytes, metav1.PatchOptions{})
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to remove finalizers from cstorvolumeconfig {%s}",
			cvc.Name,
		)
	}
	klog.Infof("finalizers removed successfully from cstorvolumeconfig {%s}", cvc.Name)
	return nil
}

// getPendingCVRCount gets the pending replica count to be created
// in case of any failures
func (c *CVCController) getPendingCVRCount(
	cvc *apis.CStorVolumeConfig,
) (int, error) {

	currentReplicaCount, err := c.getCurrentReplicaCount(cvc)
	if err != nil {
		runtime.HandleError(err)
		return 0, err
	}
	return cvc.Spec.Provision.ReplicaCount - currentReplicaCount, nil
}

// getCurrentReplicaCount give the current cstorvolumereplicas count for the
// given volume.
func (c *CVCController) getCurrentReplicaCount(cvc *apis.CStorVolumeConfig) (int, error) {
	// TODO use lister
	//	CVRs, err := c.cvrLister.CStorVolumeReplicas(cvc.Namespace).
	//		List(klabels.Set(pvLabel).AsSelector())

	pvLabel := pvSelector + "=" + cvc.Name

	cvrList, err := c.clientset.
		CstorV1().
		CStorVolumeReplicas(cvc.Namespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: pvLabel})

	if err != nil {
		return 0, errors.Errorf("unable to get current replica count: %v", err)
	}
	return len(cvrList.Items), nil
}

// IsCVRPending look for pending cstorvolume replicas compared to desired
// replica count. returns true if count doesn't matches.
func (c *CVCController) IsCVRPending(cvc *apis.CStorVolumeConfig) (bool, error) {

	selector := klabels.SelectorFromSet(BaseLabels(cvc))
	CVRs, err := c.cvrLister.CStorVolumeReplicas(cvc.Namespace).
		List(selector)
	if err != nil {
		return false, errors.Errorf("failed to list cvr : %v", err)
	}
	// TODO: check for greater values
	return cvc.Spec.Provision.ReplicaCount != len(CVRs), nil
}

// BaseLabels returns the base labels we apply to cstorvolumereplicas created
func BaseLabels(cvc *apis.CStorVolumeConfig) map[string]string {
	base := map[string]string{
		pvSelector: cvc.Name,
	}
	return base
}

// cvcNeedResize returns true if a cvc desired a resize operation.
func (c *CVCController) cvcNeedResize(cvc *apis.CStorVolumeConfig) bool {

	desiredCVCSize := cvc.Spec.Capacity[corev1.ResourceStorage]
	actualCVCSize := cvc.Status.Capacity[corev1.ResourceStorage]

	return desiredCVCSize.Cmp(actualCVCSize) > 0
}

// resizeCVC will:
// 1. Mark cvc as resizing.
// 2. Resize the cstorvolume object.
// 3. Mark cvc as resizing finished
func (c *CVCController) resizeCVC(cvc *apis.CStorVolumeConfig) error {
	var updatedCVC *apis.CStorVolumeConfig
	var err error
	cv, err := c.clientset.CstorV1().CStorVolumes(cvc.Namespace).
		Get(context.TODO(), cvc.Name, metav1.GetOptions{})
	if err != nil {
		runtime.HandleError(fmt.Errorf("falied to get cv %s: %v", cvc.Name, err))
		return err
	}
	desiredCVCSize := cvc.Spec.Capacity[corev1.ResourceStorage]

	if (cv.Spec.Capacity).Cmp(cv.Status.Capacity) > 0 {
		c.recorder.Event(cvc, corev1.EventTypeNormal, string(apis.CStorVolumeConfigResizing),
			fmt.Sprintf("Resize already in progress %s", cvc.Name))

		klog.Warningf("Resize already in progress on %q from: %v to: %v",
			cvc.Name, cv.Status.Capacity.String(), cv.Spec.Capacity.String())
		return nil
	}

	// markCVC as resized finished
	if desiredCVCSize.Cmp(cv.Status.Capacity) == 0 {
		// Resize volume succeeded mark it as resizing finished.
		return c.markCVCResizeFinished(cvc)
	}

	//if desiredCVCSize.Cmp(cv.Spec.Capacity) > 0 {
	if updatedCVC, err = c.markCVCResizeInProgress(cvc); err != nil {
		klog.Errorf("failed to mark cvc %q as resizing: %v", cvc.Name, err)
		return err
	}
	cvc = updatedCVC
	// Record an event to indicate that cvc-controller is resizing this volume.
	c.recorder.Event(cvc, corev1.EventTypeNormal, string(apis.CStorVolumeConfigResizing),
		fmt.Sprintf("CVCController is resizing volume %s", cvc.Name))

	err = c.resizeCV(cv, desiredCVCSize)
	if err != nil {
		// Record an event to indicate that resize operation is failed.
		c.recorder.Eventf(cvc, corev1.EventTypeWarning, string(apis.CStorVolumeConfigResizeFailed), err.Error())
		return err
	}
	return nil
}

func (c *CVCController) markCVCResizeInProgress(cvc *apis.CStorVolumeConfig) (*apis.CStorVolumeConfig, error) {
	// Mark CVC as Resize Started
	progressCondition := apis.CStorVolumeConfigCondition{
		Type:               apis.CStorVolumeConfigResizing,
		LastTransitionTime: metav1.Now(),
	}
	newCVC := cvc.DeepCopy()
	newCVC.Status.Conditions = MergeResizeConditionsOfCVC(newCVC.Status.Conditions,
		[]apis.CStorVolumeConfigCondition{progressCondition})
	return c.PatchCVCStatus(cvc, newCVC)
}

type resizeProcessStatus struct {
	condition apis.CStorVolumeConfigCondition
	processed bool
}

// MergeResizeConditionsOfCVC updates cvc with desired resize conditions
// leaving other conditions untouched.
func MergeResizeConditionsOfCVC(oldConditions, resizeConditions []apis.CStorVolumeConfigCondition) []apis.CStorVolumeConfigCondition {

	resizeConditionMap := map[apis.CStorVolumeConfigConditionType]*resizeProcessStatus{}

	for _, condition := range resizeConditions {
		resizeConditionMap[condition.Type] = &resizeProcessStatus{condition, false}
	}

	newConditions := []apis.CStorVolumeConfigCondition{}
	for _, condition := range oldConditions {
		// If Condition is of not resize type, we keep it.
		if _, ok := knownResizeConditions[condition.Type]; !ok {
			newConditions = append(newConditions, condition)
			continue
		}

		if newCondition, ok := resizeConditionMap[condition.Type]; ok {
			newConditions = append(newConditions, newCondition.condition)
			newCondition.processed = true
		}
	}
	// append all unprocessed conditions
	for _, newCondition := range resizeConditionMap {
		if !newCondition.processed {
			newConditions = append(newConditions, newCondition.condition)
		}
	}
	return newConditions
}

func (c *CVCController) markCVCResizeFinished(cvc *apis.CStorVolumeConfig) error {
	newCVC := cvc.DeepCopy()
	newCVC.Status.Capacity = cvc.Spec.Capacity

	newCVC.Status.Conditions = MergeResizeConditionsOfCVC(cvc.Status.Conditions, []apis.CStorVolumeConfigCondition{})
	_, err := c.PatchCVCStatus(cvc, newCVC)
	if err != nil {
		klog.Errorf("Mark CVC %q as resize finished failed: %v", cvc.Name, err)
		return err
	}

	klog.V(4).Infof("Resize CVC %q finished", cvc.Name)
	c.recorder.Eventf(cvc, corev1.EventTypeNormal, string(apis.CStorVolumeConfigResizeSuccess), "Resize volume succeeded")

	return nil
}

// PatchCVCStatus updates CVC status using patch api
func (c *CVCController) PatchCVCStatus(oldCVC,
	newCVC *apis.CStorVolumeConfig,
) (*apis.CStorVolumeConfig, error) {
	patchBytes, _, err := getPatchData(oldCVC, newCVC)
	if err != nil {
		return nil, fmt.Errorf("can't patch status of CVC %s as generate path data failed: %v", oldCVC.Name, err)
	}
	updatedClaim, updateErr := c.clientset.CstorV1().CStorVolumeConfigs(oldCVC.Namespace).
		Patch(context.TODO(), oldCVC.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})

	if updateErr != nil {
		return nil, fmt.Errorf("can't patch status of CVC %s with %v", oldCVC.Name, updateErr)
	}
	return updatedClaim, nil
}

func getPatchData(oldObj, newObj interface{}) ([]byte, []byte, error) {
	oldData, err := json.Marshal(oldObj)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal old object failed: %v", err)
	}
	newData, err := json.Marshal(newObj)
	if err != nil {
		return nil, nil, fmt.Errorf("mashal new object failed: %v", err)
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, oldObj)
	if err != nil {
		return nil, nil, fmt.Errorf("CreateTwoWayMergePatch failed: %v", err)
	}
	return patchBytes, oldData, nil
}

// resizeCV resize the cstor volume to desired size, and update CV's capacity
func (c *CVCController) resizeCV(cv *apis.CStorVolume, newCapacity resource.Quantity) error {
	newCV := cv.DeepCopy()
	newCV.Spec.Capacity = newCapacity

	patchBytes, _, err := getPatchData(cv, newCV)
	if err != nil {
		return fmt.Errorf("can't update capacity of CV %s as generate patch data failed: %v", cv.Name, err)
	}
	_, updateErr := c.clientset.CstorV1().CStorVolumes(openebsNamespace).
		Patch(context.TODO(), cv.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}

// deletePDBIfNotInUse deletes the PDB if no volume is refering to the
// cStorvolumeconfig PDB
func (c *CVCController) deletePDBIfNotInUse(cvc *apis.CStorVolumeConfig) error {
	//TODO: If HALease is enabled active-active then below code needs to be
	//revist
	pdbName := getPDBName(cvc)
	cvcLabelSelector := string(apitypes.PodDisruptionBudgetKey) + "=" + pdbName
	cvcList, err := c.clientset.CstorV1().CStorVolumeConfigs(openebsNamespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: cvcLabelSelector})
	if err != nil {
		return errors.Wrapf(err,
			"failed to list volumes refering to PDB %s", pdbName)
	}
	if len(cvcList.Items) == 1 {
		err = c.kubeclientset.PolicyV1beta1().PodDisruptionBudgets(openebsNamespace).
			Delete(context.TODO(), pdbName, metav1.DeleteOptions{})
		if k8serror.IsNotFound(err) {
			klog.Infof("pdb %s of volume %s was already deleted", pdbName, cvc.Name)
			return nil
		}
		if err != nil {
			return err
		}
		klog.Infof("Successfully deleted the PDB %s of volume %s", pdbName, cvc.Name)
	}
	return nil
}

// scaleVolumeReplicas identifies whether it is scaleup or scaledown case of
// volume replicas. If user added entry of pool info under the spec then changes
// are treated as scaleup case. If user removed poolInfo entry from spec then
// changes are treated as scale down case. If user just modifies the pool entry
// info under the spec then it is a kind of migration which is not yet supported
func (c *CVCController) scaleVolumeReplicas(cvc *apis.CStorVolumeConfig) error {
	var err error
	if len(cvc.Spec.Policy.ReplicaPoolInfo) > len(cvc.Status.PoolInfo) {
		cvc, err = c.scaleUpVolumeReplicas(cvc)
	} else if len(cvc.Spec.Policy.ReplicaPoolInfo) < len(cvc.Status.PoolInfo) {
		cvc, err = c.scaleDownVolumeReplicas(cvc)
	} else {
		c.recorder.Event(cvc, corev1.EventTypeWarning, "Migration",
			"Migration of volume replicas is not yet supported")
		return nil
	}
	if err != nil {
		c.recorder.Eventf(cvc,
			corev1.EventTypeWarning,
			"ScalingVolumeReplicas",
			"%v", err)
		return err
	}
	c.recorder.Eventf(cvc,
		corev1.EventTypeNormal,
		"ScalingVolumeReplicas",
		"successfully scaled volume replicas to %d", len(cvc.Status.PoolInfo))
	return nil
}

func (c *CVCController) ShouldReconcile(cvc *apis.CStorVolumeConfig) (bool, string) {
	cvcOperatorVersion := version.Current()
	cvcVersion := cvc.VersionDetails.Status.Current
	// if version is not exists means its a brand new resource that has not been
	// reconciled by controller yet
	if cvcVersion == "" {
		return true, ""
	}

	if cvcVersion != cvcOperatorVersion {
		return false, fmt.Sprintf("cvc operator version is %s but cvc version is %s",
			cvcOperatorVersion,
			cvcVersion)
	}
	return true, ""
}

func (c *CVCController) reconcileVersion(cvc *apis.CStorVolumeConfig) (*apis.CStorVolumeConfig, error) {
	var err error
	// the below code uses deep copy to have the state of object just before
	// any update call is done so that on failure the last state object can be returned
	if cvc.VersionDetails.Status.Current != cvc.VersionDetails.Desired {
		if !version.IsCurrentVersionValid(cvc.VersionDetails.Status.Current) {
			return cvc, errors.Errorf("invalid current version %s", cvc.VersionDetails.Status.Current)
		}
		if !version.IsDesiredVersionValid(cvc.VersionDetails.Desired) {
			return cvc, errors.Errorf("invalid desired version %s", cvc.VersionDetails.Desired)
		}
		cvcObj := cvc.DeepCopy()
		if cvc.VersionDetails.Status.State != apis.ReconcileInProgress {
			cvcObj.VersionDetails.Status.SetInProgressStatus()
			cvcObj, err = c.clientset.CstorV1().CStorVolumeConfigs(cvcObj.Namespace).Update(context.TODO(), cvcObj, metav1.UpdateOptions{})
			if err != nil {
				return cvc, err
			}
		}
		// As no other steps are required just change current version to
		// desired version
		path := strings.Split(cvcObj.VersionDetails.Status.Current, "-")[0]
		u := &upgradeParams{
			cvc:    cvcObj,
			client: c.clientset,
		}
		// Get upgrade function for corresponding path, if path does not
		// exits then no upgrade is required and funcValue will be nil.
		funcValue := upgradeMap[path]
		if funcValue != nil {
			cvcObj, err = funcValue(u)
			if err != nil {
				return cvcObj, err
			}
		}
		cvc = cvcObj.DeepCopy()
		cvcObj.VersionDetails.SetSuccessStatus()
		cvcObj, err = c.clientset.CstorV1().CStorVolumeConfigs(cvcObj.Namespace).Update(context.TODO(), cvcObj, metav1.UpdateOptions{})
		if err != nil {
			return cvc, errors.Wrap(err, "failed to update cvc")
		}
		return cvcObj, nil
	}
	return cvc, nil
}

// populateVersion assigns VersionDetails for CVC object
func (c *CVCController) populateVersion(cvc *apis.CStorVolumeConfig) (*apis.CStorVolumeConfig, error) {
	if cvc.VersionDetails.Status.Current == "" {
		version := version.Current()

		cvc.VersionDetails.Status.Current = version
		// For newly created CVC Desired field will also be empty
		cvc.VersionDetails.Desired = version
		cvc.VersionDetails.Status.DependentsUpgraded = true
		obj, err := c.clientset.CstorV1().
			CStorVolumeConfigs(cvc.Namespace).
			Update(context.TODO(), cvc, metav1.UpdateOptions{})

		if err != nil {
			return nil, errors.Wrapf(
				err,
				"failed to update cvc %s while adding versiondetails",
				cvc.Name,
			)
		}
		klog.Infof("Version %s added on cvc %s", version, cvc.Name)
		return obj, nil
	}
	return cvc, nil
}
