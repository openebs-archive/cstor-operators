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
	"time"

	cstorapis "github.com/openebs/api/pkg/apis/cstor/v1"
	"github.com/openebs/api/pkg/util"
	"github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var (
	// CVCPhaseTimeout is how long CVC requires to change the phase
	CVCPhaseTimeout = 3 * time.Minute
	// CVCDeletingTimeout is How long claims have to become deleted.
	CVCDeletingTimeout = 3 * time.Minute
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
		if util.IsChangeInLists(desiredPoolNames, cvcObj.Status.PoolInfo) {
			return nil
		}
		klog.Infof(
			"CStorVolumeConfig %s found and desired pools=%v current pools=%v (%v)",
			cvcName, desiredPoolNames, cvcObj.Status.PoolInfo, time.Since(start))
	}
	return errors.Errorf("CStorVolumeConfig %s replicas are not yet present in desired pools", cvcName)
}

// WaitForCStorVolumeConfigDeleted waits for a CStorVolumeConfig
// to be removed from the system until timeout occurs, whichever comes first
func (client *Client) WaitForCStorVolumeConfigDeleted(cvcName, cvcNamespace string, poll, timeout time.Duration) error {
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
		Get(cvcName, metav1.GetOptions{})
}
