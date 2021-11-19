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
	"encoding/json"
	"time"

	cstorapis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/klog"
)

var (
	// CVCPhaseTimeout is how long CVC requires to change the phase
	CVCPhaseTimeout = 3 * time.Minute
	// CVCDeletingTimeout is How long claims have to become deleted.
	CVCDeletingTimeout = 3 * time.Minute
	// CVCScaleTimeout is how long CVC requires to change the replica pools list
	CVCScaleTimeout = 3 * time.Minute
)

// WaitForCStorVolumeConfigPhase waits for a CStorVolumeConfig to
// be in a specific phase or until timeout occurs, whichever comes first
func (client *Client) WaitForCStorVolumeConfigPhase(
	cvcName, cvcNamespace string, expectedPhase cstorapis.CStorVolumeConfigPhase, poll, timeout time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		cvcObj, err := client.GetCVC(cvcName, cvcNamespace)
		if err != nil {
			// Currently we are returning error but based on the requirment we can retry to get CVC
			return err
		}
		if cvcObj.Status.Phase == expectedPhase {
			return nil
		}
		klog.Infof("CStorVolumeConfig %s found and phase=%s (%v)", cvcName, cvcObj.Status.Phase, time.Since(start))
	}
	return errors.Errorf("CStorVolumeConfig %s not at all in phase %s", cvcName, expectedPhase)
}

// WaitForCStorVolumeReplicaPools waits for a replicas
// to exist in desired pools untill timeout occurs, whichever comes first
func (client *Client) WaitForCStorVolumeReplicaPools(
	cvcName, cvcNamespace string, poll, timeout time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		cvcObj, err := client.GetCVC(cvcName, cvcNamespace)
		if err != nil {
			// Currently we are returning error but based on the requirment we can retry to get CVC
			return err
		}
		desiredPoolNames := cvcObj.GetDesiredReplicaPoolNames()
		if !util.IsChangeInLists(desiredPoolNames, cvcObj.Status.PoolInfo) {
			return nil
		}
		klog.Infof(
			"Waiting for CStorVolumeConfig %s to match desired pools=%v and current pools=%v (%v)",
			cvcName, desiredPoolNames, cvcObj.Status.PoolInfo, time.Since(start))
	}
	return errors.Errorf("CStorVolumeConfig %s replicas are not yet present in desired pools", cvcName)
}

// WaitForCStorVolumeConfigDeletion waits for a CStorVolumeConfig
// to be removed from the system until timeout occurs, whichever comes first
func (client *Client) WaitForCStorVolumeConfigDeletion(cvcName, cvcNamespace string, poll, timeout time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		_, err := client.GetCVC(cvcName, cvcNamespace)
		if err != nil {
			if k8serror.IsNotFound(err) {
				return nil
			}
			return err
		}
	}
	return errors.Errorf("CStorVolumeConfig %s is not removed from the system within %v", cvcName, timeout)
}

// GetCVC will fetch the CVC from etcd
func (client *Client) GetCVC(cvcName, cvcNamespace string) (*cstorapis.CStorVolumeConfig, error) {
	return client.OpenEBSClientSet.CstorV1().
		CStorVolumeConfigs(cvcNamespace).
		Get(context.TODO(), cvcName, metav1.GetOptions{})
}

// PatchCVCSpec patch the cvc object by fetching from etcd
func (client *Client) PatchCVCSpec(cvcName, cvcNamespace string,
	cvcSpec cstorapis.CStorVolumeConfigSpec) (*cstorapis.CStorVolumeConfig, error) {
	existingCVCObj, err := client.GetCVC(cvcName, cvcNamespace)
	if err != nil {
		return nil, err
	}
	cloneCVC := existingCVCObj.DeepCopy()
	cloneCVC.Spec = cvcSpec
	patchBytes, _, err := getPatchData(existingCVCObj, cloneCVC)
	if err != nil {
		return nil, err
	}
	return client.OpenEBSClientSet.CstorV1().CStorVolumeConfigs(existingCVCObj.Namespace).
		Patch(context.TODO(), existingCVCObj.Name, k8stypes.MergePatchType, patchBytes, metav1.PatchOptions{})
}

// PatchCVC patch the cvc object by fetching from etcd
func (client *Client) PatchCVC(newCVCObj *cstorapis.CStorVolumeConfig) (*cstorapis.CStorVolumeConfig, error) {
	existingCVCObj, err := client.GetCVC(newCVCObj.Name, newCVCObj.Namespace)
	if err != nil {
		return nil, err
	}
	patchBytes, _, err := getPatchData(existingCVCObj, newCVCObj)
	if err != nil {
		return nil, err
	}
	return client.OpenEBSClientSet.CstorV1().CStorVolumeConfigs(existingCVCObj.Namespace).
		Patch(context.TODO(), existingCVCObj.Name, k8stypes.MergePatchType, patchBytes, metav1.PatchOptions{})
}

func getPatchData(oldObj, newObj interface{}) ([]byte, []byte, error) {
	oldData, err := json.Marshal(oldObj)
	if err != nil {
		return nil, nil, errors.Errorf("marshal old object failed: %v", err)
	}
	newData, err := json.Marshal(newObj)
	if err != nil {
		return nil, nil, errors.Errorf("mashal new object failed: %v", err)
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, oldObj)
	if err != nil {
		return nil, nil, errors.Errorf("CreateTwoWayMergePatch failed: %v", err)
	}
	return patchBytes, oldData, nil
}
