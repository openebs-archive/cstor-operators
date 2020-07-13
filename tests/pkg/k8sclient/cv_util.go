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
	"reflect"
	"time"

	cstorapis "github.com/openebs/api/pkg/apis/cstor/v1"
	openebstypes "github.com/openebs/api/pkg/apis/types"
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
	// CVDeletingTimeout is How log CV have to become deleted.
	CVDeletingTimeout = 5 * time.Minute
)

// WaitForCStorVolumePhase waits for a CStorVolume to
// be in a specific phase or until timeout occurs, whichever comes first
func (client *Client) WaitForCStorVolumePhase(
	cvcName, cvcNamespace string, expectedPhase cstorapis.CStorVolumePhase, poll, timeout time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		cvcObj, err := client.GetCV(cvcName, cvcNamespace)
		if err != nil {
			// Currently we are returning error but based on the requirment we can retry to get CV
			return err
		}
		if cvcObj.Status.Phase == expectedPhase {
			return nil
		}
		klog.Infof("CStorVolume %s found and phase=%s (%v)", cvcName, cvcObj.Status.Phase, time.Since(start))
	}
	return errors.Errorf("CStorVolume %s not at all in phase %s", cvcName, expectedPhase)
}

// WaitForCStorVolumeDeleted waits for a CStorVolume
// to be removed from the system until timeout occurs, whichever comes first
func (client *Client) WaitForCStorVolumeDeleted(cvName, cvNamespace string, poll, timeout time.Duration) error {
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
func (client *Client) WaitForVolumeManagerCountEventually(name, namespace string, expectedCount int, poll, timeout time.Duration) error {
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
	return errors.Errorf("VolumeManager of %s not in expected count %d", name, expectedCount)
}

// WaitForVolumeManagerResourceLimits verifies whether resource limits are reconciled or not
func (client *Client) WaitForVolumeManagerResourceLimits(
	name, namespace string, resourceLimits, auxresourceLimits *corev1.ResourceRequirements, timeout, pool time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(pool) {
		matchedCount := 0
		targetPod, err := client.KubeClientSet.CoreV1().
			Pods(namespace).
			Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		// In target pod there were three containers and
		// cstor-istgt is the main container
		for _, container := range targetPod.Spec.Containers {
			if container.Name == "cstor-istgt" {
				if reflect.DeepEqual(container.Resources, resourceLimits) {
					matchedCount++
				}
			} else {
				if reflect.DeepEqual(container.Resources, auxresourceLimits) {
					matchedCount++
				}
			}
		}
		if matchedCount == 3 {
			return nil
		}
	}
	return errors.Errorf("VolumeManager of %s not has resource limits", name)
}

// GetCV will fetch the CV from etcd
func (client *Client) GetCV(cvcName, cvcNamespace string) (*cstorapis.CStorVolume, error) {
	return client.OpenEBSClientSet.CstorV1().
		CStorVolumes(cvcNamespace).
		Get(cvcName, metav1.GetOptions{})
}

// GetVolumeManagerList will fetch volume manager list based on provided arguments from etcd
func (client *Client) GetVolumeManagerList(name, namespace string) (*corev1.PodList, error) {
	return client.KubeClientSet.CoreV1().
		Pods(namespace).
		List(metav1.ListOptions{LabelSelector: openebstypes.PersistentVolumeLabelKey + "=" + name})
}
