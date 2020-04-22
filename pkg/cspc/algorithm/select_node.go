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

package algorithm

import (
	"fmt"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	openebsio "github.com/openebs/api/pkg/apis/openebs.io/v1alpha1"
	"github.com/openebs/api/pkg/apis/types"
	"github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	unit = 1024
)

// SelectNode returns a node where pool should be created.
func (ac *Config) SelectNode() (*cstor.PoolSpec, string, error) {
	usedNodes, err := ac.GetUsedNodes()
	if err != nil {
		return nil, "", errors.Wrapf(err, "could not get used nodes list for pool creation")
	}
	for _, pool := range ac.CSPC.Spec.Pools {
		// pin it
		pool := pool
		nodeName, err := ac.GetNodeFromLabelSelector(pool.NodeSelector)
		if err != nil || nodeName == "" {
			klog.Errorf("could not use node for selectors {%v}: {%s}", pool.NodeSelector, err.Error())
			continue
		}
		if ac.VisitedNodes[nodeName] {
			continue
		} else {
			ac.VisitedNodes[nodeName] = true

			if !usedNodes[nodeName] {
				return &pool, nodeName, nil
			}
		}

	}
	return nil, "", errors.New("no node qualified for pool creation")
}

// GetNodeFromLabelSelector returns the node name selected by provided labels
func (ac *Config) GetNodeFromLabelSelector(labels map[string]string) (string, error) {
	nodeList, err := ac.kubeclientset.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: getLabelSelectorString(labels)})
	if err != nil {
		return "", errors.Wrap(err, "failed to get node list from the node selector")
	}
	if len(nodeList.Items) != 1 {
		return "", errors.Errorf("invalid no.of nodes %d from the given node selectors", len(nodeList.Items))
	}
	return nodeList.Items[0].GetLabels()[string(types.HostNameLabelKey)], nil
}

// getLabelSelectorString returns a string of label selector form label map to be used in
// list options.
func getLabelSelectorString(selector map[string]string) string {
	var selectorString string
	for key, value := range selector {
		selectorString = selectorString + key + "=" + value + ","
	}
	selectorString = selectorString[:len(selectorString)-len(",")]
	return selectorString
}

// GetUsedNode returns a map of node for which pool has already been created.
func (ac *Config) GetUsedNodes() (map[string]bool, error) {
	usedNode := make(map[string]bool)
	cspList, err := ac.
		clientset.
		CstorV1().
		CStorPoolInstances(ac.Namespace).
		List(metav1.ListOptions{LabelSelector: string(types.CStorPoolClusterLabelKey) + "=" + ac.CSPC.Name})

	if err != nil {
		return nil, errors.Wrap(err, "could not list already created csp(s)")
	}
	for _, cspObj := range cspList.Items {
		usedNode[cspObj.Labels[string(types.HostNameLabelKey)]] = true
	}
	return usedNode, nil
}

// GetBDListForNode returns a list of BD from the pool spec.
func GetBDListForNode(pool cstor.PoolSpec) []string {
	var BDList []string
	for _, group := range pool.DataRaidGroups {
		for _, bd := range group.CStorPoolInstanceBlockDevices {
			BDList = append(BDList, bd.BlockDeviceName)
		}
	}
	return BDList
}

// ClaimBDsForNode claims a given BlockDevice for node
// If the block device(s) is/are already claimed for any other CSPC it returns error.
// If the block device(s) is/are already calimed for the same CSPC -- it is left as it is and can be used for
// pool provisioning.
// If the block device(s) is/are unclaimed, then those are claimed.
func (ac *Config) ClaimBDsForNode(BD []string) error {
	pendingClaim := 0
	pendingClaimBDs := make(map[string]bool)
	for _, bdName := range BD {
		bdAPIObj, err := ac.clientset.OpenebsV1alpha1().BlockDevices(ac.Namespace).Get(bdName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "error in getting details for BD {%s} whether it is claimed", bdName)
		}

		if IsBlockDeviceClaimed(*bdAPIObj) {
			IsClaimedBDUsable, errBD := ac.IsClaimedBDUsable(*bdAPIObj)
			if errBD != nil {
				return errors.Wrapf(errBD, "error in getting details for BD {%s} for usability", bdName)
			}

			if !IsClaimedBDUsable {
				return errors.Errorf("BD {%s} already in use", bdName)
			}
			continue
		}

		err = ac.ClaimBD(*bdAPIObj)
		if err != nil {
			return errors.Wrapf(err, "Failed to claim BD {%s}", bdName)
		}
		pendingClaimBDs[bdAPIObj.Name] = true
		pendingClaim++
	}

	if pendingClaim > 0 {
		return errors.Errorf("%d block device claims are pending:%v", pendingClaim, pendingClaimBDs)
	}
	return nil
}

// ClaimBD claims a given BlockDevice
func (ac *Config) ClaimBD(bdObj openebsio.BlockDevice) error {
	resourceList, err := GetCapacity(ByteCount(bdObj.Spec.Capacity.Storage))
	if err != nil {
		return errors.Errorf("failed to get capacity from block device %s:%s", bdObj.Name, err)
	}

	newBDCObj := openebsio.NewBlockDeviceClaim().
		WithName("bdc-cstor-" + string(bdObj.UID)).
		WithNamespace(ac.Namespace).
		WithLabels(map[string]string{types.CStorPoolClusterLabelKey: ac.CSPC.Name}).
		WithBlockDeviceName(bdObj.Name).
		WithHostName(bdObj.Labels[types.HostNameLabelKey]).
		WithCSPCOwnerReference(GetCSPCOwnerReference(ac.CSPC)).
		WithCapacity(resourceList).
		WithFinalizer(types.CSPCFinalizer)

	_, err = ac.clientset.OpenebsV1alpha1().BlockDeviceClaims(ac.Namespace).Create(newBDCObj)
	if k8serror.IsAlreadyExists(err) {
		klog.Infof("BDC for BD {%s} already created", bdObj.Name)
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to create block device claim for bd {%s}", bdObj.Name)
	}
	return nil
}

// IsClaimedBDUsable returns true if the passed BD is already claimed and can be
// used for provisioning
func (ac *Config) IsClaimedBDUsable(bd openebsio.BlockDevice) (bool, error) {
	if IsBlockDeviceClaimed(bd) {
		claimRef := bd.Spec.ClaimRef
		if claimRef == nil {
			return false, errors.New("nil claim reference found in bd")
		}

		bdcName := claimRef.Name
		bdcAPIObject, err := ac.clientset.OpenebsV1alpha1().BlockDeviceClaims(ac.Namespace).Get(bdcName, metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "could not get block device claim for block device {%s}", bd.Name)
		}
		if BDCHasLabel(types.CStorPoolClusterLabelKey, ac.CSPC.Name, *bdcAPIObject) {
			return true, nil
		}
	} else {
		return false, errors.Errorf("block device {%s} is not claimed", bd.Name)
	}
	return false, nil
}

// IsBlockDeviceClaimed returns true if the provided block devie is claimed.
func IsBlockDeviceClaimed(bd openebsio.BlockDevice) bool {
	return bd.Status.ClaimState == openebsio.BlockDeviceClaimed
}

// BDCHasLabel returns true if the provided key,value exists as label on block device claim.
func BDCHasLabel(labelKey, labelValue string, bdc openebsio.BlockDeviceClaim) bool {
	val, ok := bdc.GetLabels()[labelKey]
	if ok {
		return val == labelValue
	}
	return false
}

func GetCapacity(capacity string) (resource.Quantity, error) {
	resCapacity, err := resource.ParseQuantity(capacity)
	if err != nil {
		return resource.Quantity{}, errors.Errorf("Failed to parse capacity:{%s}", err.Error())
	}
	return resCapacity, nil
}

// ByteCount converts bytes into corresponding unit
func ByteCount(b uint64) string {
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, index := uint64(unit), 0
	for val := b / unit; val >= unit; val /= unit {
		div *= unit
		index++
	}
	return fmt.Sprintf("%d%c",
		uint64(b)/uint64(div), "KMGTPE"[index])
}
