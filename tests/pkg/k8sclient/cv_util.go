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

package k8sclient

import (
	"context"
	"reflect"
	"strconv"
	"time"

	cstorapis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebstypes "github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var (
	// CVPhaseTimeout is how long CV have to reach particular state.
	CVPhaseTimeout = 5 * time.Minute
	// VolumeManagerPhaseTimeout is how long volume manger to reach particular state
	VolumeManagerPhaseTimeout = 5 * time.Minute
	// CVDeletingTimeout is How long CV have to become deleted.
	CVDeletingTimeout = 5 * time.Minute
	// VolumeMangerConfigTimeout is How long volume manager will take reach specified configuration changes
	VolumeMangerConfigTimeout = 5 * time.Minute
	// CVReplicaConnectionTimeout is How long CV have to wait for replicas to register
	CVReplicaConnectionTimeout = 5 * time.Minute
)

// WaitForCStorVolumePhase waits for a CStorVolume to
// be in a specific phase or until timeout occurs, whichever comes first
func (client *Client) WaitForCStorVolumePhase(
	cvName, cvNamespace string, expectedPhase cstorapis.CStorVolumePhase, poll, timeout time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		cvcObj, err := client.GetCV(cvName, cvNamespace)
		if err != nil {
			// Currently we are returning error but based on the requirment we can retry to get CV
			return err
		}
		if cvcObj.Status.Phase == expectedPhase {
			return nil
		}
		klog.Infof("CStorVolume %s found and phase=%s (%v)", cvName, cvcObj.Status.Phase, time.Since(start))
	}
	return errors.Errorf("CStorVolume %s not at all in phase %s", cvName, expectedPhase)
}

// WaitForDesiredReplicaConnections waits for a desired replicas
// to connect to CStorVolume or until timeout occurs, whichever comes first
func (client *Client) WaitForDesiredReplicaConnections(
	cvName, cvNamespace string, desiredReplicaIDs []string, poll, timeout time.Duration) error {
	replicationFactor := len(desiredReplicaIDs)
	consistencyFactor := replicationFactor/2 + 1
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		cvObj, err := client.GetCV(cvName, cvNamespace)
		if err != nil {
			// Currently we are returning error but based on the requirment we can retry to get CV
			return err
		}
		if len(desiredReplicaIDs) != len(cvObj.Status.ReplicaDetails.KnownReplicas) {
			klog.Infof("Waiting for %d replicas to available on CStorVolume %s but got %d",
				len(desiredReplicaIDs), cvObj.Name,
				len(cvObj.Status.ReplicaDetails.KnownReplicas))
			continue
		}
		currentReplicaIds := []string{}
		for replicaID := range cvObj.Status.ReplicaDetails.KnownReplicas {
			currentReplicaIds = append(currentReplicaIds, string(replicaID))
		}
		if util.IsChangeInLists(desiredReplicaIDs, currentReplicaIds) {
			return errors.Errorf("CStorVolume doesn't has specified replica IDs")
		}
		if replicationFactor != cvObj.Spec.ReplicationFactor {
			return errors.Errorf("CStorVolume RF %d is not matching with expected RF %d",
				replicationFactor, cvObj.Spec.ReplicationFactor)
		}
		if consistencyFactor != cvObj.Spec.ConsistencyFactor {
			return errors.Errorf("CStorVolume CF %d is not matching with expected CF %d",
				consistencyFactor, cvObj.Spec.ConsistencyFactor)
		}
		return nil
	}
	return errors.Errorf("CStorVolume %s doesn't have desired replica IDs timeout occured", cvName)
}

// WaitForCStorVolumeDeletion waits for a CStorVolume
// to be removed from the system until timeout occurs, whichever comes first
func (client *Client) WaitForCStorVolumeDeletion(cvName, cvNamespace string, poll, timeout time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		_, err := client.GetCV(cvName, cvNamespace)
		if err != nil {
			if k8serror.IsNotFound(err) {
				return nil
			}
			return err
		}
	}
	return errors.Errorf("CStorVolume %s is not removed from the system within %v", cvName, timeout)
}

// WaitForVolumeManagerCountEventually gets the volume-manager deployment count based on cv
func (client *Client) WaitForVolumeManagerCountEventually(
	name, namespace string, expectedCount int, poll, timeout time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		vList, err := client.GetVolumeManagerList(name, namespace)
		if err != nil {
			return err
		}
		if len(vList.Items) == expectedCount {
			return nil
		}
		klog.Infof("Volme manager %s found and count=%d (%v)", name, len(vList.Items), time.Since(start))
	}
	return errors.Errorf("Volume manager of %s not in expected count %d", name, expectedCount)
}

// WaitForVolumeManagerTolerations will wait for volume manager to have
// tolerations or untill timeout occurs, which ever comes first
func (client *Client) WaitForVolumeManagerTolerations(
	name, namespace string, tolerations []corev1.Toleration, timeout, poll time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		vDeploymentList, err := client.KubeClientSet.AppsV1().Deployments(namespace).
			List(context.TODO(), metav1.ListOptions{LabelSelector: openebstypes.PersistentVolumeLabelKey + "=" + name})
		if err != nil {
			return err
		}
		if len(vDeploymentList.Items) != 1 {
			return errors.Errorf("found %d no.of deployments", len(vDeploymentList.Items))
		}
		areTolerationsExist := false
		for _, toleration := range tolerations {
			isExist := false
			for _, deployTolerations := range vDeploymentList.Items[0].Spec.Template.Spec.Tolerations {
				if reflect.DeepEqual(toleration, deployTolerations) {
					isExist = true
					break
				}
			}
			if !isExist {
				areTolerationsExist = false
				break
			}
			areTolerationsExist = true
		}
		// In case if custom tolerations are removed by default CVC-Operator is adding 4 tolerations
		// Ref: https://github.com/openebs/cstor-operators/blob/9e7d5b7c64bdfafd0d3127f044706072da7c78d6/pkg/controllers/cstorvolumeconfig/deployment.go#L164
		if len(tolerations) == 0 && len(vDeploymentList.Items[0].Spec.Template.Spec.Tolerations) == 4 {
			areTolerationsExist = true
		}

		if areTolerationsExist {
			return nil
		}

		klog.Infof("Waiting for tolerations to propogate to volume manager deployment %s since (%v)", name, time.Since(start))
	}
	return errors.Errorf("Volume manager deployment %s doesn't specified tolerations", name)
}

// WaitForVolumeManagerPriorityClass will wait for volume manager to have
// priorityclass name or untill timeout occurs, whichever comes first
func (client *Client) WaitForVolumeManagerPriorityClass(
	name, namespace, priorityClassName string, timeout, poll time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		vList, err := client.GetVolumeManagerList(name, namespace)
		if err != nil && !k8serror.IsNotFound(err) {
			return err
		}
		// Pod might me deleted and created back due to changes in
		// deployment
		if k8serror.IsNotFound(err) {
			continue
		}
		for _, targetPod := range vList.Items {
			if !targetPod.DeletionTimestamp.IsZero() {
				continue
			}
			if targetPod.Spec.PriorityClassName == priorityClassName {
				return nil
			}
		}
		klog.Infof("Waiting for priority class name to propogated to volume manager %s since (%v)", name, time.Since(start))
	}
	return errors.Errorf("Volume manager %s doesn't have specified prority class name", name)
}

// WaitForVolumeManagerNodeSelector will wait for volume manager to have
// node selector changes or untill timeout occurs, whichever comes first
func (client *Client) WaitForVolumeManagerNodeSelector(
	name, namespace string, nodeLabels map[string]string, timeout, poll time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		vList, err := client.GetVolumeManagerList(name, namespace)
		if err != nil && !k8serror.IsNotFound(err) {
			return err
		}
		// Pod might me deleted and created back due to changes in
		// deployment
		if k8serror.IsNotFound(err) {
			continue
		}
		for _, targetPod := range vList.Items {
			if !targetPod.DeletionTimestamp.IsZero() {
				continue
			}
			if reflect.DeepEqual(targetPod.Spec.NodeSelector, nodeLabels) {
				return nil
			}
		}
		klog.Infof("Waiting for node selector to be propogated to volume manager %s since (%v)", name, time.Since(start))
	}
	return errors.Errorf("Volume manager %s doesn't have specified node selector values", name)
}

// WaitForVolumeManagerResourceLimits waits for volume manager to have
// resource limits or untill timeout occurs, whichever comes first
func (client *Client) WaitForVolumeManagerResourceLimits(
	name, namespace string, resourceLimits, auxresourceLimits corev1.ResourceRequirements, timeout, poll time.Duration) error {
	// auxresource will be present in both side car so doubling the count of auxulary resources
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		isResourceRequirmentsMatched := true
		// isVisited will be marked true if it iterated over containers
		isVisited := false

		vList, err := client.GetVolumeManagerList(name, namespace)
		if err != nil && !k8serror.IsNotFound(err) {
			return err
		}
		// Pod might me deleted and created back due to changes in
		// deployment
		if k8serror.IsNotFound(err) {
			continue
		}

		// In target pod there were three containers and
		// cstor-istgt is the main container
		for _, targetPod := range vList.Items {
			if !targetPod.DeletionTimestamp.IsZero() {
				continue
			}
			isVisited = true
			for _, container := range targetPod.Spec.Containers {
				if container.Name == "cstor-istgt" {
					// Checking for resource limits
					isMatched := isResourceListsMatched(container.Resources.Limits, resourceLimits.Limits)
					if !isMatched {
						isResourceRequirmentsMatched = false
					}

					// Checking for resource requests
					isMatched = isResourceListsMatched(container.Resources.Requests, resourceLimits.Requests)
					if !isMatched {
						isResourceRequirmentsMatched = false
					}
				} else {
					isMatched := isResourceListsMatched(container.Resources.Limits, auxresourceLimits.Limits)
					if !isMatched {
						isResourceRequirmentsMatched = false
					}
					isMatched = isResourceListsMatched(container.Resources.Requests, auxresourceLimits.Requests)
					if !isMatched {
						isResourceRequirmentsMatched = false
					}
				}
			}
		}
		if isVisited && isResourceRequirmentsMatched {
			return nil
		}
		klog.Infof("Waiting for resource limits to propogate to volume manager %s since (%v)",
			name, time.Since(start))
	}
	return errors.Errorf("Volume manager %s doesn't not have resource limits", name)
}

// WaitForVolumeManagerTunables will wait for volume manager to have
// specified tunables or untill timeout occurs, whichever comes first
func (client *Client) WaitForVolumeManagerTunables(
	name, namespace string, luWorkers int64, queueDepth string, timeout, poll time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		vList, err := client.GetVolumeManagerList(name, namespace)
		if err != nil && !k8serror.IsNotFound(err) {
			return err
		}
		// Pod might me deleted and created back due to changes in
		// deployment
		if k8serror.IsNotFound(err) {
			continue
		}
		isMatched := false
		for _, targetPod := range vList.Items {
			if !targetPod.DeletionTimestamp.IsZero() {
				continue
			}
			for _, container := range targetPod.Spec.Containers {
				if container.Name == "cstor-istgt" {
					for _, envVar := range container.Env {
						if envVar.Name == "Luworkers" {
							existingVal, err := strconv.Atoi(envVar.Value)
							if err != nil {
								errors.Wrapf(err, "failed to parse %s value of luworkers", envVar.Value)
							}
							if int64(existingVal) != luWorkers {
								isMatched = false
								break
							}
							isMatched = true
						}
						if envVar.Name == "QueDeepth" {
							if envVar.Value != queueDepth {
								isMatched = false
								break
							}
							isMatched = true
						}
					}
				}
			}
		}
		if isMatched {
			return nil
		}
		klog.Infof("Waiting for tunables to propogated to volume manager %s since (%v)", name, time.Since(start))
	}
	return errors.Errorf("Volume manager %s doesn't have specified tunables", name)
}

// isResourceListsMatched returns true if provided resource are matched else false
func isResourceListsMatched(resourceListOne, resourceListTwo corev1.ResourceList) bool {
	isMatched := true

	for resourceName, existingQuantity := range resourceListOne {
		expectedQuantity := resourceListTwo[resourceName]
		if expectedQuantity.Cmp(existingQuantity) != 0 {
			isMatched = false
		}
	}

	for resourceName, existingQuantity := range resourceListTwo {
		expectedQuantity := resourceListOne[resourceName]
		if expectedQuantity.Cmp(existingQuantity) != 0 {
			isMatched = false
		}
	}
	return isMatched
}

// GetCV will fetch the CV from etcd
func (client *Client) GetCV(cvName, cvNamespace string) (*cstorapis.CStorVolume, error) {
	return client.OpenEBSClientSet.CstorV1().
		CStorVolumes(cvNamespace).
		Get(context.TODO(), cvName, metav1.GetOptions{})
}

// GetVolumeManagerList will fetch volume manager list based on provided arguments from etcd
func (client *Client) GetVolumeManagerList(name, namespace string) (*corev1.PodList, error) {
	return client.KubeClientSet.CoreV1().
		Pods(namespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: openebstypes.PersistentVolumeLabelKey + "=" + name})
}
