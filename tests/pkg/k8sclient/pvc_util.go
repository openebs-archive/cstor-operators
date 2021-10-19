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
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var (
	// ClaimBindingTimeout is how long claims have to become bound.
	ClaimBindingTimeout = 3 * time.Minute
	// ClaimDeletingTimeout is How long claims have to become deleted.
	ClaimDeletingTimeout = 3 * time.Minute
)

// CreateNamespace create namespace for volume
func (client *Client) CreateNamespace(ns string) error {
	_, err := client.KubeClientSet.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		if k8serror.IsNotFound(err) {
			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			}
			_, err = client.KubeClientSet.CoreV1().Namespaces().Create(context.TODO(), nsObj, metav1.CreateOptions{})
		}
	}
	return err
}

// WaitForPersistentVolumeClaimPhase waits for a PersistentVolumeClaim to
// be in a specific phase or until timeout occurs, whichever comes first
func (client *Client) WaitForPersistentVolumeClaimPhase(
	pvcName, pvcNamespace string, expectedPhase corev1.PersistentVolumeClaimPhase, poll, timeout time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		pvcObj, err := client.GetPVC(pvcName, pvcNamespace)
		if err != nil {
			// Currently we are returning error but based on the requirment we can retry to get PVC
			return err
		}
		if pvcObj.Status.Phase == expectedPhase {
			return nil
		}
		klog.Infof("PersistentVolumeClaim %s found and phase=%s (%v)", pvcName, pvcObj.Status.Phase, time.Since(start))
	}
	return errors.Errorf("PersistentVolumeClaim %s not at all in phase %s", pvcName, expectedPhase)
}

// GetPVC will fetch the PVC from etcd
func (client *Client) GetPVC(pvcName, pvcNamespace string) (*corev1.PersistentVolumeClaim, error) {
	return client.KubeClientSet.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
}

// WaitForPersistentVolumeClaimDeletion waits for a PersistentVolumeClaim
// to be removed from the system until timeout occurs, whichever comes first
func (client *Client) WaitForPersistentVolumeClaimDeletion(pvcName, pvcNamespace string, poll, timeout time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		_, err := client.GetPVC(pvcName, pvcNamespace)
		if err != nil {
			if k8serror.IsNotFound(err) {
				return nil
			}
			return err
		}
		klog.Infof("Waiting for %s pvc in %s namespace to be deleted since(%v)", pvcName, pvcNamespace, time.Since(start))
	}
	return errors.Errorf("PersistentVolumeClaim %s is not removed from the system within %v", pvcName, timeout)
}
