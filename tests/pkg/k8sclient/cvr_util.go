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

	cstorapis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebstypes "github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var (
	// CVRPhaseTimeout is how long CVR have to reach particular state.
	CVRPhaseTimeout = 5 * time.Minute
)

// WaitForCVRCountEventually waits for a CStorVolumeReplicas to
// be in a specific phase or until timeout occurs, whichever comes first
func (client *Client) WaitForCVRCountEventually(
	name, namespace string, expectedCount int, poll, timeout time.Duration, predicateList ...cstorapis.CVRPredicate) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		cvrList, err := client.GetCVRList(name, namespace)
		if err != nil {
			// Currently we are returning error but based on the requirment we can retry to get PVC
			return err
		}
		filteredList := cvrList.Filter(predicateList...)
		if len(filteredList.Items) == expectedCount {
			return nil
		}
		klog.Infof("Waiting for %d CStorVolumeReplias to exist in expected state but got %d", expectedCount, len(filteredList.Items))
	}
	return errors.Errorf("Expected count %d of CStorVolumeReplicas are not availbe for volume %s", expectedCount, name)
}

// VerifyCVRPoolNames will verify whether volumes replicas are provisioned in desired pools or not
func (client *Client) VerifyCVRPoolNames(name, namespace string, poolNames []string) error {
	cvrList, err := client.GetCVRList(name, namespace)
	if err != nil {
		return err
	}
	if util.IsChangeInLists(poolNames, cvrList.GetPoolNames()) {
		return errors.Errorf("One/more CStorVolumeReplicas are not in pool names %v", poolNames)
	}
	return nil
}

// GetCVRReplicaIDs return list of replicaIDs of replicas of provided Volume
func (client *Client) GetCVRReplicaIDs(name, namespace string) ([]string, error) {
	cvrList, err := client.GetCVRList(name, namespace)
	if err != nil {
		return nil, err
	}
	replicaIDs := make([]string, len(cvrList.Items))
	for i, cvrObj := range cvrList.Items {
		replicaIDs[i] = cvrObj.Spec.ReplicaID
	}
	return replicaIDs, nil
}

// GetCVRList will fetch the CVRList from etcd
func (client *Client) GetCVRList(pvName, pvcNamespace string) (*cstorapis.CStorVolumeReplicaList, error) {
	return client.OpenEBSClientSet.CstorV1().
		CStorVolumeReplicas(pvcNamespace).
		List(context.TODO(), metav1.ListOptions{
			LabelSelector: openebstypes.PersistentVolumeLabelKey + "=" + pvName,
		})
}
