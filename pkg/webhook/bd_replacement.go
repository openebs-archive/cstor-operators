/*
Copyright 2019 The OpenEBS Authors.

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

package webhook

import (
	"fmt"
	"reflect"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/pkg/apis/openebs.io/v1alpha1"
	"github.com/openebs/api/pkg/apis/types"
	clientset "github.com/openebs/api/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//TODO: Update BlockDeviceReplacement to generic name

// BlockDeviceReplacement contains old and new CSPC to validate for block device replacement
type BlockDeviceReplacement struct {
	// OldCSPC is the persisted CSPC in etcd.
	OldCSPC *cstor.CStorPoolCluster
	// NewCSPC is the CSPC after it has been modified but yet not persisted to etcd.
	NewCSPC *cstor.CStorPoolCluster
	// kubeClient is a standard kubernetes clientset
	kubeClient kubernetes.Interface

	// clientset is a openebs custom resource package generated for custom API group.
	clientset clientset.Interface
}

// NewBlockDeviceReplacement returns an empty BlockDeviceReplacement object.
func NewBlockDeviceReplacement(k kubernetes.Interface, c clientset.Interface) *BlockDeviceReplacement {
	return &BlockDeviceReplacement{
		OldCSPC:    &cstor.CStorPoolCluster{},
		NewCSPC:    &cstor.CStorPoolCluster{},
		kubeClient: k,
		clientset:  c,
	}
}

// WithOldCSPC sets the old persisted CSPC into the BlockDeviceReplacement object.
func (bdr *BlockDeviceReplacement) WithOldCSPC(oldCSPC *cstor.CStorPoolCluster) *BlockDeviceReplacement {
	bdr.OldCSPC = oldCSPC
	return bdr
}

// WithNewCSPC sets the new CSPC as a result of CSPC modification which is not yet persisted,
// into the BlockDeviceReplacement object
func (bdr *BlockDeviceReplacement) WithNewCSPC(newCSPC *cstor.CStorPoolCluster) *BlockDeviceReplacement {
	bdr.NewCSPC = newCSPC
	return bdr
}

type poolspecs struct {
	oldSpec []cstor.PoolSpec
	newSpec []cstor.PoolSpec
}

// ValidateSpecChanges validates the changes in CSPC for changes in a raid group only if the
// update/edit of CSPC can trigger a block device replacement/pool expansion
// scenarios.
func ValidateSpecChanges(commonPoolSpecs *poolspecs, bdr *BlockDeviceReplacement) (bool, string) {
	for i, oldPoolSpec := range commonPoolSpecs.oldSpec {
		oldPoolSpec := oldPoolSpec
		// process only when there is change in pool specs
		if reflect.DeepEqual(&oldPoolSpec, &commonPoolSpecs.newSpec[i]) {
			continue
		}
		if ok, msg := bdr.IsPoolSpecChangeValid(&oldPoolSpec, &commonPoolSpecs.newSpec[i]); !ok {
			return false, msg
		}
	}
	return true, ""
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

// GetNodeFromLabelSelector returns the node name selected by provided labels
func GetNodeFromLabelSelector(labels map[string]string, kubeClient kubernetes.Interface) (string, error) {
	nodeList, err := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: getLabelSelectorString(labels)})
	if err != nil {
		return "", errors.Wrap(err, "failed to get node list from the node selector")
	}
	if len(nodeList.Items) != 1 {
		return "", errors.Errorf("invalid no.of nodes %d from the given node selectors", len(nodeList.Items))
	}
	return nodeList.Items[0].GetLabels()[types.HostNameLabelKey], nil
}

// getCommonPoolSpecs get the same pool specs from old persisted CSPC and the new CSPC after modification
// which is not persisted yet.
func getCommonPoolSpecs(cspcNew, cspcOld *cstor.CStorPoolCluster, kubeClient kubernetes.Interface) (*poolspecs, error) {
	commonPoolSpecs := &poolspecs{
		oldSpec: []cstor.PoolSpec{},
		newSpec: []cstor.PoolSpec{},
	}
	for _, oldPool := range cspcOld.Spec.Pools {
		oldNodeName, err := GetNodeFromLabelSelector(oldPool.NodeSelector, kubeClient)
		if err != nil {
			return nil, err
		}

		for _, newPool := range cspcNew.Spec.Pools {
			newNodeName, err := GetNodeFromLabelSelector(newPool.NodeSelector, kubeClient)
			if err != nil {
				return nil, err
			}
			if oldNodeName == newNodeName {
				commonPoolSpecs.oldSpec = append(commonPoolSpecs.oldSpec, oldPool)
				commonPoolSpecs.newSpec = append(commonPoolSpecs.newSpec, newPool)
				break
			}
		}
	}
	return commonPoolSpecs, nil
}

// validateRaidGroupChanges returns error when user removes or add block
// devices(for other than strip type) to existing raid group or else it will
// return nil
func validateRaidGroupChanges(oldRg, newRg *cstor.RaidGroup) error {
	// // return error when block devices are removed from new raid group
	// if len(newRg.BlockDevices) < len(oldRg.BlockDevices) {
	// 	return errors.Errorf("removing block device from %s raid group is not valid operation",
	// 		oldRg.Type)
	// }
	// // return error when block device are added to new raid group other than
	// // stripe
	// if cstor.PoolType(oldRg.Type) != cstor.PoolStriped &&
	// 	len(newRg.BlockDevices) > len(oldRg.BlockDevices) {
	// 	return errors.Errorf("adding block devices to existing %s raid group is "+
	// 		"not valid operation",
	// 		oldRg.Type)
	// }
	return nil
}

// IsPoolSpecChangeValid validates the pool specs on CSPC for raid groups
// changes case
func (bdr *BlockDeviceReplacement) IsPoolSpecChangeValid(oldPoolSpec, newPoolSpec *cstor.PoolSpec) (bool, string) {
	newToOldBd := make(map[string]string)
	for _, oldRg := range oldPoolSpec.DataRaidGroups {
		oldRg := oldRg // pin it
		isRaidGroupExist := false
		// if oldRg.Type == "" {
		// 	oldRg.Type = oldPoolSpec.PoolConfig.DefaultRaidGroupType
		// }
		for _, newRg := range newPoolSpec.DataRaidGroups {
			newRg := newRg // pin it
			if IsRaidGroupCommon(oldRg, newRg) {
				isRaidGroupExist = true
				if err := validateRaidGroupChanges(&oldRg, &newRg); err != nil {
					return false, fmt.Sprintf("raid group validation failed: %v", err)
				}
				if IsBlockDeviceReplacementCase(&oldRg, &newRg) {
					if ok, msg := bdr.IsBDReplacementValid(&newRg, &oldRg); !ok {
						return false, msg
					}
					newBD := GetNewBDFromRaidGroups(&newRg, &oldRg)
					for k, v := range newBD {
						newToOldBd[k] = v
					}
				}
				break
			}
		}
		// Old raid group should exist on new pool spec changes
		if !isRaidGroupExist {
			return false, fmt.Sprintf("removing raid group from pool spec is invalid operation")
		}
	}

	for newBD, oldBD := range newToOldBd {
		err := bdr.createBDC(newBD, oldBD)
		if err != nil {
			return false, err.Error()
		}
	}
	return true, ""
}

// IsRaidGroupCommon returns true if the provided raid groups are the same raid groups.
func IsRaidGroupCommon(rgOld, rgNew cstor.RaidGroup) bool {
	oldBdMap := make(map[string]bool)
	for _, oldBD := range rgOld.BlockDevices {
		oldBdMap[oldBD.BlockDeviceName] = true
	}

	for _, newBD := range rgNew.BlockDevices {
		if oldBdMap[newBD.BlockDeviceName] {
			return true
		}
	}
	return false
}

// IsBlockDeviceReplacementCase returns true if the edit/update of CSPC can trigger a blockdevice
// replacement.
func IsBlockDeviceReplacementCase(newRaidGroup, oldRaidGroup *cstor.RaidGroup) bool {
	count := GetNumberOfDiskReplaced(newRaidGroup, oldRaidGroup)
	return count >= 1
}

// GetNumberOfDiskReplaced returns the nuber of disk replaced in raid group.
func GetNumberOfDiskReplaced(newRG, oldRG *cstor.RaidGroup) int {
	var count int
	oldBlockDevicesMap := make(map[string]bool)
	for _, bdOld := range oldRG.BlockDevices {
		oldBlockDevicesMap[bdOld.BlockDeviceName] = true
	}
	for _, newBD := range newRG.BlockDevices {
		if !oldBlockDevicesMap[newBD.BlockDeviceName] {
			count++
		}
	}
	return count
}

// IsBDReplacementValid validates for BD replacement.
func (bdr *BlockDeviceReplacement) IsBDReplacementValid(newRG, oldRG *cstor.RaidGroup) (bool, string) {

	// if oldRG.Type == string(cstor.PoolStriped) {
	// 	return false, "cannot replace  blockdevice in stripe raid group"
	// }

	// Not more than 1 bd should be replaced in a raid group.
	if IsMoreThanOneDiskReplaced(newRG, oldRG) {
		return false, "cannot replace more than one blockdevice in a raid group"
	}

	// The incoming BD for replacement should not be present in the current CSPC.
	if bdr.IsNewBDPresentOnCurrentCSPC(newRG, oldRG) {
		return false, "the new blockdevice intended to use for replacement is already a part of the current cspc"
	}

	// No background replacement should be going on in the raid group undergoing replacement.
	if ok, err := bdr.IsExistingReplacmentInProgress(oldRG); ok {
		return false, fmt.Sprintf("cannot replace blockdevice as a "+
			"background replacement may be in progress in the raid group: %s", err.Error())
	}

	// The incoming BD should be a valid entry if
	// 1. The BD does not have a BDC.
	// 2. The BD has a BDC with the current CSPC label and there is no successor of this BD
	//    present in the CSPC.
	if !bdr.AreNewBDsValid(newRG, oldRG, bdr.OldCSPC) {
		return false, "the new blockdevice intended to use for replacement in invalid"
	}

	if err := bdr.validateNewBDCapacity(newRG, oldRG); err != nil {
		return false, fmt.Sprintf("error: %v", err)
	}

	return true, ""
}

// validateNewBDCapacity returns error only when new block device has less capacity
// than existing block device
func (bdr *BlockDeviceReplacement) validateNewBDCapacity(newRG, oldRG *cstor.RaidGroup) error {
	newToOldBlockDeviceMap := GetNewBDFromRaidGroups(newRG, oldRG)
	bdClient := bdr.clientset.OpenebsV1alpha1().BlockDevices(bdr.OldCSPC.Namespace)
	for newBDName, oldBDName := range newToOldBlockDeviceMap {
		newBDObj, err := bdClient.Get(newBDName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to get capacity of replaced block device: %s", newBDName)
		}
		oldBDObj, err := bdClient.Get(oldBDName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to get capacity of existing block device: %s", oldBDName)
		}
		if newBDObj.Spec.Capacity.Storage < oldBDObj.Spec.Capacity.Storage {
			return errors.Errorf("capacity of replacing block device {%s:%d} "+
				"should be greater than or equal to existing block device {%s:%d}",
				newBDName, newBDObj.Spec.Capacity.Storage,
				oldBDName, oldBDObj.Spec.Capacity.Storage)
		}
	}
	return nil
}

// IsMoreThanOneDiskReplaced returns true if more than one disk is replaced in the same raid group.
func IsMoreThanOneDiskReplaced(newRG, oldRG *cstor.RaidGroup) bool {
	count := GetNumberOfDiskReplaced(newRG, oldRG)
	return count > 1
}

// IsNewBDPresentOnCurrentCSPC returns true if the new/incoming BD that will be used for replacement
// is already present in CSPC.
func (bdr *BlockDeviceReplacement) IsNewBDPresentOnCurrentCSPC(newRG, oldRG *cstor.RaidGroup) bool {
	newBDs := GetNewBDFromRaidGroups(newRG, oldRG)
	for _, pool := range bdr.OldCSPC.Spec.Pools {
		for _, rg := range pool.DataRaidGroups {
			for _, bd := range rg.BlockDevices {
				if _, ok := newBDs[bd.BlockDeviceName]; ok {
					return true
				}
			}
		}
	}
	return false
}

// IsExistingReplacmentInProgress returns true if a block device in raid group is under active replacement.
func (bdr *BlockDeviceReplacement) IsExistingReplacmentInProgress(oldRG *cstor.RaidGroup) (bool, error) {
	for _, v := range oldRG.BlockDevices {
		bdcObject, err := bdr.GetBDCOfBD(v.BlockDeviceName)
		if err != nil {
			return true, errors.Errorf("failed to query for any existing replacement in the raid group : %s", err.Error())
		}
		_, ok := bdcObject.GetAnnotations()[types.PredecessorBDLabelKey]
		if ok {
			return true, errors.Errorf("replacement is still in progress for bd %s", v.BlockDeviceName)
		}
	}
	return false, nil
}

// AreNewBDsValid returns true if the new BDs are valid BDs for replacement.
func (bdr *BlockDeviceReplacement) AreNewBDsValid(newRG, oldRG *cstor.RaidGroup, oldcspc *cstor.CStorPoolCluster) bool {
	newBDs := GetNewBDFromRaidGroups(newRG, oldRG)
	for bd := range newBDs {
		bdc, err := bdr.GetBDCOfBD(bd)
		if err != nil {
			return false
		}
		if !bdr.IsBDValid(bd, bdc, oldcspc) {
			return false
		}
	}
	return true
}

// IsBDValid returns true if the new BD is a valid BD for replacement.
func (bdr *BlockDeviceReplacement) IsBDValid(bd string, bdc *openebsapis.BlockDeviceClaim, oldcspc *cstor.CStorPoolCluster) bool {
	if bdc != nil && bdc.GetLabels()[types.CStorPoolClusterLabelKey] != oldcspc.Name {
		return false
	}
	predecessorMap, err := bdr.GetPredecessorBDIfAny(oldcspc)
	if err != nil {
		return false
	}
	if predecessorMap[bd] {
		return false
	}
	return true
}

// GetPredecessorBDIfAny returns a map of predecessor BDs if any in the current CSPC
// Note: Predecessor BDs in a CSPC are those BD for which a new BD has appeared in the CSPC and
//       replacement is still in progress
//
// For example,
// (b1,b2) is a group in cspc
// which has been changed to ( b3,b2 )  [Notice that b1 got replaced by b3],
// now b1 is not present in CSPC but the replacement is still in progress in background.
// In this case b1 is a predecessor BD.
func (bdr *BlockDeviceReplacement) GetPredecessorBDIfAny(cspcOld *cstor.CStorPoolCluster) (map[string]bool, error) {
	predecessorBDMap := make(map[string]bool)
	for _, pool := range cspcOld.Spec.Pools {
		for _, rg := range pool.DataRaidGroups {
			for _, bd := range rg.BlockDevices {
				bdc, err := bdr.GetBDCOfBD(bd.BlockDeviceName)
				if err != nil {
					return nil, err
				}
				if bdc == nil {
					continue
				}
				predecessorBDMap[bdc.GetAnnotations()[types.PredecessorBDLabelKey]] = true
			}
		}
	}
	return predecessorBDMap, nil
}

// GetBDCOfBD returns the BDC object for corresponding BD.
func (bdr *BlockDeviceReplacement) GetBDCOfBD(bdName string) (*openebsapis.BlockDeviceClaim, error) {
	bdcList, err := bdr.clientset.OpenebsV1alpha1().BlockDeviceClaims(bdr.OldCSPC.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.Errorf("failed to list bdc: %s", err.Error())
	}
	list := []openebsapis.BlockDeviceClaim{}
	for _, bdc := range bdcList.Items {
		if bdc.Spec.BlockDeviceName == bdName {
			list = append(list, bdc)
		}
	}

	// If there is not BDC for a BD -- this means it an acceptable situation for BD replacement
	// The incoming BD finally will have a BDC created, hence no error is returned.
	if len(list) == 0 {
		return nil, nil
	}

	if len(list) != 1 {
		return nil, errors.Errorf("did not get exact one bdc for the bd %s", bdName)
	}
	return &list[0], nil
}

func (bdr *BlockDeviceReplacement) createBDC(newBD, oldBD string) error {
	bdObj, err := bdr.clientset.OpenebsV1alpha1().BlockDevices(bdr.OldCSPC.Namespace).Get(newBD, v1.GetOptions{})
	if err != nil {
		return err
	}
	err = bdr.ClaimBD(bdObj, oldBD)
	if err != nil {
		return err
	}
	return nil
}

func getBDOwnerReference(cspc *cstor.CStorPoolCluster) []metav1.OwnerReference {
	OwnerReference := []metav1.OwnerReference{
		*metav1.NewControllerRef(cspc, cstor.SchemeGroupVersion.WithKind("CStorPoolCluster")),
	}
	return OwnerReference
}

// ClaimBD claims a given BlockDevice
func (bdr *BlockDeviceReplacement) ClaimBD(newBdObj *openebsapis.BlockDevice, oldBD string) error {
	newBDCObj := openebsapis.NewBlockDeviceClaim().
		WithName("bdc-cstor-" + string(newBdObj.UID)).
		WithNamespace(newBdObj.Namespace).
		WithLabels(map[string]string{types.CStorPoolClusterLabelKey: bdr.OldCSPC.Name}).
		WithAnnotations(map[string]string{types.PredecessorBDLabelKey: oldBD}).
		WithBlockDeviceName(newBdObj.Name).
		WithHostName(newBdObj.Labels[types.HostNameLabelKey]).
		WithCapacity(resource.MustParse(ByteCount(newBdObj.Spec.Capacity.Storage))).
		WithCSPCOwnerReference(getBDOwnerReference(bdr.OldCSPC)[0]).
		WithFinalizer(types.CSPCFinalizer)

	bdcClient := bdr.clientset.OpenebsV1alpha1().BlockDeviceClaims(newBdObj.Namespace)
	bdcObj, err := bdcClient.Get(newBDCObj.Name, v1.GetOptions{})
	if k8serror.IsNotFound(err) {
		_, err = bdcClient.Create(newBDCObj)
		if err != nil {
			return errors.Wrapf(err, "failed to create block device claim for bd {%s}", newBdObj.Name)
		}
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "failed to get block device claim for bd {%s}", newBdObj.Name)
	}

	bdcObj.WithAnnotations(map[string]string{types.PredecessorBDLabelKey: oldBD})
	if err != nil {
		return errors.Wrapf(err, "failed to add annotation on block device claim {%s}", bdcObj.Name)
	}

	_, err = bdcClient.
		Update(bdcObj)
	return err
}

// GetNewBDFromRaidGroups returns a map of new successor bd to old bd for replacement in a raid group
func GetNewBDFromRaidGroups(newRG, oldRG *cstor.RaidGroup) map[string]string {
	newToOldBlockDeviceMap := make(map[string]string)
	oldBlockDevicesMap := make(map[string]bool)
	newBlockDevicesMap := make(map[string]bool)

	for _, bdOld := range oldRG.BlockDevices {
		oldBlockDevicesMap[bdOld.BlockDeviceName] = true
	}

	for _, bdNew := range newRG.BlockDevices {
		newBlockDevicesMap[bdNew.BlockDeviceName] = true
	}
	var newBD, oldBD string

	for _, newRG := range newRG.BlockDevices {
		if !oldBlockDevicesMap[newRG.BlockDeviceName] {
			newBD = newRG.BlockDeviceName
			break
		}
	}

	for _, oldRG := range oldRG.BlockDevices {
		if !newBlockDevicesMap[oldRG.BlockDeviceName] {
			oldBD = oldRG.BlockDeviceName
			break
		}
	}
	newToOldBlockDeviceMap[newBD] = oldBD
	return newToOldBlockDeviceMap
}
