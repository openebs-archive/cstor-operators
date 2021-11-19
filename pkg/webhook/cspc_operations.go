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

package webhook

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	"github.com/openebs/api/v3/pkg/apis/types"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	dataRG       = "data"
	writeCacheRG = "writeCache"
)

// PoolOperations contains old and new CSPC to validate for pool
// operations
type PoolOperations struct {
	// OldCSPC is the persisted CSPC in etcd.
	OldCSPC *cstor.CStorPoolCluster
	// NewCSPC is the CSPC after it has been modified but yet not persisted to etcd.
	NewCSPC *cstor.CStorPoolCluster
	// kubeClient is a standard kubernetes clientset
	kubeClient kubernetes.Interface

	// clientset is a openebs custom resource package generated for custom API group.
	clientset clientset.Interface
}

// NewPoolOperations returns an empty PoolOperations object.
func NewPoolOperations(k kubernetes.Interface, c clientset.Interface) *PoolOperations {
	return &PoolOperations{
		kubeClient: k,
		clientset:  c,
	}
}

// WithOldCSPC sets the old persisted CSPC into the PoolOperations object.
func (pOps *PoolOperations) WithOldCSPC(oldCSPC *cstor.CStorPoolCluster) *PoolOperations {
	pOps.OldCSPC = oldCSPC
	return pOps
}

// WithNewCSPC sets the new CSPC as a result of CSPC modification which is not yet persisted,
// into the PoolOperations object
func (pOps *PoolOperations) WithNewCSPC(newCSPC *cstor.CStorPoolCluster) *PoolOperations {
	pOps.NewCSPC = newCSPC
	return pOps
}

type poolspecs struct {
	oldSpec []cstor.PoolSpec
	newSpec []cstor.PoolSpec
}

// ValidateSpecChanges validates the changes in CSPC for changes in a raid group only if the
// update/edit of CSPC can trigger a block device replacement/pool expansion
// scenarios.
func ValidateSpecChanges(commonPoolSpecs *poolspecs, pOps *PoolOperations) (bool, string) {
	for i, oldPoolSpec := range commonPoolSpecs.oldSpec {
		oldPoolSpec := oldPoolSpec
		// process only when there is change in pool specs
		if reflect.DeepEqual(&oldPoolSpec, &commonPoolSpecs.newSpec[i]) {
			continue
		}
		if ok, msg := pOps.ArePoolSpecChangesValid(&oldPoolSpec, &commonPoolSpecs.newSpec[i]); !ok {
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

// GetHostNameFromLabelSelector returns the node name selected by provided labels
func GetHostNameFromLabelSelector(labels map[string]string, kubeClient kubernetes.Interface) (string, error) {
	nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: getLabelSelectorString(labels)})
	if err != nil {
		return "", errors.Wrap(err, "failed to get node list from the node selector")
	}
	if len(nodeList.Items) != 1 {
		return "", errors.Errorf("invalid no.of nodes %d from the given node selectors", len(nodeList.Items))
	}
	return nodeList.Items[0].GetLabels()[types.HostNameLabelKey], nil
}

// getCommonPoolSpecs get the same pool specs from old persisted CSPC and the new CSPC after modification
// which is not persisted yet. It figures out common pool spec using following ways
// 1. If node exist in cluster for current node selector then common pool
//    spec will be figured out using nodeselector.
// 2. If node doesn't exist in cluster for current node selector then
//    common pool spec will be figured using data raid groups blockdevices.
// NOTE: First check is more priority to avoid blockdevice replacement in case of stripe pool
// TODO: Fix cases where node and blockdevice were replaced at a time
func getCommonPoolSpecs(cspcNew, cspcOld *cstor.CStorPoolCluster, kubeClient kubernetes.Interface) (*poolspecs, error) {
	commonPoolSpecs := &poolspecs{
		oldSpec: []cstor.PoolSpec{},
		newSpec: []cstor.PoolSpec{},
	}
	for _, oldPoolSpec := range cspcOld.Spec.Pools {
		// isNodeExist helps to get common pool spec based on nodeSelector
		isNodeExist := true

		var oldNodeName string
		nodeList, err := kubeClient.CoreV1().
			Nodes().
			List(context.TODO(), metav1.ListOptions{LabelSelector: getLabelSelectorString(oldPoolSpec.NodeSelector)})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get node list from the node selector")
		}
		// If more than one node exist for given node selector
		if len(nodeList.Items) > 1 {
			return nil, errors.Errorf(
				"invalid no.of nodes %d from the given node selectors: %v",
				len(nodeList.Items), oldPoolSpec.NodeSelector)
		} else if len(nodeList.Items) == 0 {
			klog.Warningf("node doesn't exist for given nodeselector: %v", oldPoolSpec.NodeSelector)
			isNodeExist = false
		} else {
			oldNodeName = nodeList.Items[0].Name
		}

		for _, newPoolSpec := range cspcNew.Spec.Pools {
			if isNodeExist {
				newNodeName, err := GetHostNameFromLabelSelector(newPoolSpec.NodeSelector, kubeClient)
				if err != nil {
					return nil, err
				}
				if oldNodeName == newNodeName {
					commonPoolSpecs.oldSpec = append(commonPoolSpecs.oldSpec, oldPoolSpec)
					commonPoolSpecs.newSpec = append(commonPoolSpecs.newSpec, newPoolSpec)
					break
				}
			} else {
				// add into spec even if one blockdevice matches
				if hasCommonDataBlockDevicce(oldPoolSpec, newPoolSpec) {
					commonPoolSpecs.oldSpec = append(commonPoolSpecs.oldSpec, oldPoolSpec)
					commonPoolSpecs.newSpec = append(commonPoolSpecs.newSpec, newPoolSpec)
					break
				}
			}
		}
	}
	return commonPoolSpecs, nil
}

// hasCommonDataBlockDevice will return true if old and new pool spec has
// atleast one common data blockdevice
func hasCommonDataBlockDevicce(oldPoolSpec, newPoolSpec cstor.PoolSpec) bool {
	bdMap := map[string]bool{}
	for _, oldRG := range oldPoolSpec.DataRaidGroups {
		for _, cspiBD := range oldRG.CStorPoolInstanceBlockDevices {
			bdMap[cspiBD.BlockDeviceName] = true
		}
	}

	for _, newRG := range newPoolSpec.DataRaidGroups {
		for _, cspiBD := range newRG.CStorPoolInstanceBlockDevices {
			if bdMap[cspiBD.BlockDeviceName] {
				return true
			}
		}
	}
	return false
}

// validateRaidGroupChanges returns error when user removes or add block
// devices(for other than strip type) to existing raid group or else it will
// return nil
func validateRaidGroupChanges(oldRg, newRg *cstor.RaidGroup, oldRgType string) error {
	// return error when block devices are removed from new raid group
	if len(newRg.CStorPoolInstanceBlockDevices) < len(oldRg.CStorPoolInstanceBlockDevices) {
		return errors.Errorf("removing block device from %s raid group is not valid operation",
			oldRgType)
	}
	// return error when block device are added to new raid group other than
	// stripe
	if cstor.PoolType(oldRgType) != cstor.PoolStriped &&
		len(newRg.CStorPoolInstanceBlockDevices) > len(oldRg.CStorPoolInstanceBlockDevices) {
		return errors.Errorf("adding block devices to existing %s raid group is "+
			"not valid operation",
			oldRgType)
	}
	return nil
}

// IsRaidGroupCommon returns true if the provided raid groups are the same raid groups.
func IsRaidGroupCommon(rgOld, rgNew cstor.RaidGroup) bool {
	oldBdMap := make(map[string]bool)
	for _, oldBD := range rgOld.CStorPoolInstanceBlockDevices {
		oldBdMap[oldBD.BlockDeviceName] = true
	}

	for _, newBD := range rgNew.CStorPoolInstanceBlockDevices {
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
	for _, bdOld := range oldRG.CStorPoolInstanceBlockDevices {
		oldBlockDevicesMap[bdOld.BlockDeviceName] = true
	}
	for _, newBD := range newRG.CStorPoolInstanceBlockDevices {
		if !oldBlockDevicesMap[newBD.BlockDeviceName] {
			count++
		}
	}
	return count
}

// IsBDReplacementValid validates for BD replacement.
func (pOps *PoolOperations) IsBDReplacementValid(newRG, oldRG *cstor.RaidGroup, oldRgType string) (bool, string) {

	if oldRgType == string(cstor.PoolStriped) {
		return false, "cannot replace  blockdevice in stripe raid group"
	}

	// Not more than 1 bd should be replaced in a raid group.
	if IsMoreThanOneDiskReplaced(newRG, oldRG) {
		return false, "cannot replace more than one blockdevice in a raid group"
	}

	// The incoming BD for replacement should not be present in the current CSPC.
	if pOps.IsNewBDPresentOnCurrentCSPC(newRG, oldRG) {
		return false, "the new blockdevice intended to use for replacement is already a part of the current cspc"
	}

	// No background replacement should be going on in the raid group undergoing replacement.
	if ok, err := pOps.IsExistingReplacmentInProgress(oldRG); ok {
		return false, fmt.Sprintf("cannot replace blockdevice as a "+
			"background replacement may be in progress in the raid group: %s", err.Error())
	}

	// The incoming BD should be a valid entry if
	// 1. The BD does not have a BDC.
	// 2. The BD has a BDC with the current CSPC label and there is no successor of this BD
	//    present in the CSPC.
	if !pOps.AreNewBDsValid(newRG, oldRG, pOps.OldCSPC) {
		return false, "the new blockdevice intended to use for replacement in invalid"
	}

	if err := pOps.validateNewBDCapacity(newRG, oldRG); err != nil {
		return false, fmt.Sprintf("error: %v", err)
	}

	return true, ""
}

// validateNewBDCapacity returns error only when new block device has less capacity
// than existing block device
func (pOps *PoolOperations) validateNewBDCapacity(newRG, oldRG *cstor.RaidGroup) error {
	newToOldBlockDeviceMap := GetNewBDFromRaidGroups(newRG, oldRG)
	bdClient := pOps.clientset.OpenebsV1alpha1().BlockDevices(pOps.OldCSPC.Namespace)
	for newBDName, oldBDName := range newToOldBlockDeviceMap {
		newBDObj, err := bdClient.Get(context.TODO(), newBDName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to get capacity of replaced block device: %s", newBDName)
		}
		oldBDObj, err := bdClient.Get(context.TODO(), oldBDName, metav1.GetOptions{})
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
func (pOps *PoolOperations) IsNewBDPresentOnCurrentCSPC(newRG, oldRG *cstor.RaidGroup) bool {
	newBDs := GetNewBDFromRaidGroups(newRG, oldRG)
	for _, pool := range pOps.OldCSPC.Spec.Pools {
		rgs := append(pool.DataRaidGroups, pool.WriteCacheRaidGroups...)
		for _, rg := range rgs {
			for _, bd := range rg.CStorPoolInstanceBlockDevices {
				if _, ok := newBDs[bd.BlockDeviceName]; ok {
					return true
				}
			}
		}
	}
	return false
}

// IsExistingReplacmentInProgress returns true if a block device in raid group is under active replacement.
func (pOps *PoolOperations) IsExistingReplacmentInProgress(oldRG *cstor.RaidGroup) (bool, error) {
	for _, v := range oldRG.CStorPoolInstanceBlockDevices {
		bdcObject, err := pOps.GetBDCOfBD(v.BlockDeviceName)
		if err != nil {
			return true, errors.Errorf("failed to query for any existing replacement in the raid group : %s", err.Error())
		}
		if bdcObject != nil {
			_, ok := bdcObject.GetAnnotations()[types.PredecessorBDLabelKey]
			if ok {
				return true, errors.Errorf("replacement is still in progress for bd %s", v.BlockDeviceName)
			}
		}
	}
	return false, nil
}

// AreNewBDsValid returns true if the new BDs are valid BDs for replacement.
func (pOps *PoolOperations) AreNewBDsValid(newRG, oldRG *cstor.RaidGroup, oldcspc *cstor.CStorPoolCluster) bool {
	newBDs := GetNewBDFromRaidGroups(newRG, oldRG)
	for bd := range newBDs {
		bdc, err := pOps.GetBDCOfBD(bd)
		if err != nil {
			return false
		}
		if !pOps.IsBDValid(bd, bdc, oldcspc) {
			return false
		}
	}
	return true
}

// IsBDValid returns true if the new BD is a valid BD for replacement.
func (pOps *PoolOperations) IsBDValid(bd string, bdc *openebsapis.BlockDeviceClaim, oldcspc *cstor.CStorPoolCluster) bool {
	if bdc != nil && bdc.GetLabels()[types.CStorPoolClusterLabelKey] != oldcspc.Name {
		return false
	}
	predecessorMap, err := pOps.GetPredecessorBDIfAny(oldcspc)
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
func (pOps *PoolOperations) GetPredecessorBDIfAny(cspcOld *cstor.CStorPoolCluster) (map[string]bool, error) {
	predecessorBDMap := make(map[string]bool)
	for _, pool := range cspcOld.Spec.Pools {
		rgs := append(pool.DataRaidGroups, pool.WriteCacheRaidGroups...)
		for _, rg := range rgs {
			for _, bd := range rg.CStorPoolInstanceBlockDevices {
				bdc, err := pOps.GetBDCOfBD(bd.BlockDeviceName)
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
func (pOps *PoolOperations) GetBDCOfBD(bdName string) (*openebsapis.BlockDeviceClaim, error) {
	bdcList, err := pOps.clientset.OpenebsV1alpha1().BlockDeviceClaims(pOps.OldCSPC.Namespace).List(context.TODO(), v1.ListOptions{})
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

func (pOps *PoolOperations) createBDC(newBD, oldBD string) error {
	bdObj, err := pOps.clientset.OpenebsV1alpha1().BlockDevices(pOps.OldCSPC.Namespace).Get(context.TODO(), newBD, v1.GetOptions{})
	if err != nil {
		return err
	}
	return pOps.ClaimBD(bdObj, oldBD)
}

func getBDOwnerReference(cspc *cstor.CStorPoolCluster) []metav1.OwnerReference {
	OwnerReference := []metav1.OwnerReference{
		*metav1.NewControllerRef(cspc, cstor.SchemeGroupVersion.WithKind("CStorPoolCluster")),
	}
	return OwnerReference
}

// ClaimBD claims a given BlockDevice
// ToDo: The BD Claim functionality has code repetition.
// Need to think about packaging and refactor.
func (pOps *PoolOperations) ClaimBD(newBdObj *openebsapis.BlockDevice, oldBD string) error {

	// If the BD has a BD tag present then we need to decide whether
	// cStor can use it or not.
	// If there is not BD tag present on BD then still the BD is safe to use.
	value, ok := newBdObj.Labels[types.BlockDeviceTagLabelKey]
	var allowedBDTags map[string]bool
	if ok {
		// If the BD tag value is empty -- cStor cannot use it.
		if strings.TrimSpace(value) == "" {
			return errors.Errorf("failed to create block device "+
				"claim for bd {%s} as it has empty value for bd tag", newBdObj.Name)
		}

		// If the BD tag in the BD is present in allowed annotations on CSPC then
		// it means that this BD can be considered in provisioning else it should not
		// be considered
		allowedBDTags = getAllowedTagMap(pOps.NewCSPC.GetAnnotations())
		if !allowedBDTags[strings.TrimSpace(value)] {
			return errors.Errorf("cannot use bd {%s} as it has tag %s but "+
				"cspc has allowed bd tags as %s",
				newBdObj.Name, value, pOps.NewCSPC.GetAnnotations()[types.OpenEBSAllowedBDTagKey])
		}
	}

	newBDCObj := openebsapis.NewBlockDeviceClaim().
		WithName("bdc-cstor-" + string(newBdObj.UID)).
		WithNamespace(newBdObj.Namespace).
		WithLabels(map[string]string{types.CStorPoolClusterLabelKey: pOps.OldCSPC.Name}).
		WithAnnotations(map[string]string{types.PredecessorBDLabelKey: oldBD}).
		WithBlockDeviceName(newBdObj.Name).
		WithHostName(newBdObj.Labels[types.HostNameLabelKey]).
		WithCapacity(resource.MustParse(ByteCount(newBdObj.Spec.Capacity.Storage))).
		WithCSPCOwnerReference(getBDOwnerReference(pOps.OldCSPC)[0]).
		WithFinalizer(types.CSPCFinalizer)
	// ToDo: Move this to openebs/api builder
	// Create label selector to fill in BDC spec.
	if ok {
		ls := &metav1.LabelSelector{
			MatchLabels: map[string]string{types.BlockDeviceTagLabelKey: value},
		}
		newBDCObj.Spec.Selector = ls
	}

	bdcClient := pOps.clientset.OpenebsV1alpha1().BlockDeviceClaims(newBdObj.Namespace)
	bdcObj, err := bdcClient.Get(context.TODO(), newBDCObj.Name, v1.GetOptions{})
	if k8serror.IsNotFound(err) {
		_, err = bdcClient.Create(context.TODO(), newBDCObj, metav1.CreateOptions{})
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
		Update(context.TODO(), bdcObj, metav1.UpdateOptions{})
	return err
}

// getAllowedTagMap returns a map of the allowed BD tags
// Example :
// If the CSPC annotation is passed and following is the BD tag annotation
//
// cstor.openebs.io/allowed-bd-tags:fast,slow
//
// Then, a map {"fast":true,"slow":true} is returned.
func getAllowedTagMap(cspcAnnotation map[string]string) map[string]bool {
	allowedTagsMap := make(map[string]bool)
	allowedTags := cspcAnnotation[types.OpenEBSAllowedBDTagKey]
	if strings.TrimSpace(allowedTags) == "" {
		return allowedTagsMap
	}
	allowedTagsList := strings.Split(allowedTags, ",")
	for _, v := range allowedTagsList {
		if strings.TrimSpace(v) == "" {
			continue
		}
		allowedTagsMap[v] = true
	}
	return allowedTagsMap
}

// GetNewBDFromRaidGroups returns a map of new successor bd to old bd for replacement in a raid group
func GetNewBDFromRaidGroups(newRG, oldRG *cstor.RaidGroup) map[string]string {
	newToOldBlockDeviceMap := make(map[string]string)
	oldBlockDevicesMap := make(map[string]bool)
	newBlockDevicesMap := make(map[string]bool)

	for _, bdOld := range oldRG.CStorPoolInstanceBlockDevices {
		oldBlockDevicesMap[bdOld.BlockDeviceName] = true
	}

	for _, bdNew := range newRG.CStorPoolInstanceBlockDevices {
		newBlockDevicesMap[bdNew.BlockDeviceName] = true
	}
	var newBD, oldBD string

	for _, newRG := range newRG.CStorPoolInstanceBlockDevices {
		if !oldBlockDevicesMap[newRG.BlockDeviceName] {
			newBD = newRG.BlockDeviceName
			break
		}
	}

	for _, oldRG := range oldRG.CStorPoolInstanceBlockDevices {
		if !newBlockDevicesMap[oldRG.BlockDeviceName] {
			oldBD = oldRG.BlockDeviceName
			break
		}
	}
	newToOldBlockDeviceMap[newBD] = oldBD
	return newToOldBlockDeviceMap
}

// raidGroups contains list of oldraid groups and newraid groups
type raidGroups struct {
	oldRaidGroups []cstor.RaidGroup
	newRaidGroups []cstor.RaidGroup
	rgType        string
}

func getNewBDsFromStripeSpec(oldRg, newRg cstor.RaidGroup) []string {
	mapOldBlockDevices := map[string]bool{}
	bds := []string{}
	for _, bd := range oldRg.CStorPoolInstanceBlockDevices {
		mapOldBlockDevices[bd.BlockDeviceName] = true
	}
	for _, bd := range newRg.CStorPoolInstanceBlockDevices {
		if !mapOldBlockDevices[bd.BlockDeviceName] {
			bds = append(bds, bd.BlockDeviceName)
		}
	}
	return bds
}

// getBDsFromRaidGroups return list of blockdevices from list of raid groups
func getBDsFromRaidGroups(rgs []cstor.RaidGroup) []string {
	bds := []string{}
	for _, rg := range rgs {
		for _, bd := range rg.CStorPoolInstanceBlockDevices {
			bds = append(bds, bd.BlockDeviceName)
		}
	}
	return bds
}

// getExpandedRaidGroups returns the raid groups only if pool expansion is done
// by the users
func getExpandedRaidGroups(
	raidGroups []cstor.RaidGroup, oldRaidGroups []cstor.RaidGroup) []cstor.RaidGroup {
	expandedRgs := []cstor.RaidGroup{}
	for _, newRg := range raidGroups {
		isRaidGroupExist := false
		for _, oldRg := range oldRaidGroups {
			if IsRaidGroupCommon(oldRg, newRg) {
				isRaidGroupExist = true
				break
			}
		}
		// If no common blockdevices exists in raid groups then newRg is
		// added for expansion
		if !isRaidGroupExist {
			expandedRgs = append(expandedRgs, newRg)
		}
	}
	return expandedRgs
}

// validateNewBDs returns nil if new BDs are valid for expansion or else it will
// return error if occurs
func (pOps *PoolOperations) validateNewBDs(newBDs []string, cspc *cstor.CStorPoolCluster) error {
	for _, bd := range newBDs {
		bdc, err := pOps.GetBDCOfBD(bd)
		if err != nil {
			return errors.Wrapf(err, "failed to get claim of block device %s", bd)
		}
		// The incoming BD should be a valid entry if
		// 1. The BD does not have a BDC.
		// 2. The BD has a BDC with the current CSPC label and there is no successor of this BD
		//    present in the CSPC.
		if !pOps.IsBDValid(bd, bdc, cspc) {
			return errors.Errorf("can not use blockdevice %s validation failed", bd)
		}
	}
	return nil
}

// validatePoolExpansion will validate only expanded raid groups or new blockdevices(
// in stripe only block devices are added). Following are the validations:
// 1. New blockdevice shouldn't be claimed by any other CSPC (or) third party.
// 2. New blockdevice shouldn't be the replacing blockdevice.
func (pOps *PoolOperations) validatePoolExpansion(
	newPoolSpec *cstor.PoolSpec, commonRaidGroups map[string]*raidGroups) error {
	var bds []string
	for rgType, rgs := range commonRaidGroups {
		if rgs.rgType == string(cstor.PoolStriped) {
			bds = append(bds, getNewBDsFromStripeSpec(rgs.oldRaidGroups[0],
				rgs.newRaidGroups[0])...)
		} else {
			if rgType == dataRG {
				newRgs := getExpandedRaidGroups(newPoolSpec.DataRaidGroups, rgs.oldRaidGroups)
				bds = getBDsFromRaidGroups(newRgs)
			}
			if rgType == writeCacheRG {
				newRgs := getExpandedRaidGroups(newPoolSpec.WriteCacheRaidGroups, rgs.oldRaidGroups)
				bds = getBDsFromRaidGroups(newRgs)
			}
		}
	}
	return pOps.validateNewBDs(bds, pOps.OldCSPC)
}

// getIndexedCommonRaidGroups returns raidGroups that contains index matching of
// oldRaidGroups and newRaidGroups. If oldRaidGroup doesn't exist on
// newRaidGroup then return error
func getIndexedCommonRaidGroups(oldPoolSpec,
	newPoolSpec *cstor.PoolSpec) (map[string]*raidGroups, error) {
	rgs := map[string]*raidGroups{
		dataRG: &raidGroups{
			oldRaidGroups: []cstor.RaidGroup{},
			newRaidGroups: []cstor.RaidGroup{},
			rgType:        oldPoolSpec.PoolConfig.DataRaidGroupType,
		},
		writeCacheRG: &raidGroups{
			oldRaidGroups: []cstor.RaidGroup{},
			newRaidGroups: []cstor.RaidGroup{},
			rgType:        oldPoolSpec.PoolConfig.WriteCacheGroupType,
		},
	}
	// build raidGroups by identifying common raidGroups in old and new
	for _, oldRg := range oldPoolSpec.DataRaidGroups {
		isRaidGroupExist := false
		for _, newRg := range newPoolSpec.DataRaidGroups {
			if IsRaidGroupCommon(oldRg, newRg) {
				isRaidGroupExist = true
				rgs[dataRG].oldRaidGroups = append(rgs[dataRG].oldRaidGroups, oldRg)
				rgs[dataRG].newRaidGroups = append(rgs[dataRG].newRaidGroups, newRg)
				break
			}
		}
		// Old raid group should exist on new pool spec changes
		if !isRaidGroupExist {
			return nil, errors.Errorf("removing raid group from pool spec is invalid operation")
		}
	}
	for _, oldRg := range oldPoolSpec.WriteCacheRaidGroups {
		isRaidGroupExist := false
		for _, newRg := range newPoolSpec.WriteCacheRaidGroups {
			if IsRaidGroupCommon(oldRg, newRg) {
				isRaidGroupExist = true
				rgs[writeCacheRG].oldRaidGroups = append(rgs[writeCacheRG].oldRaidGroups, oldRg)
				rgs[writeCacheRG].newRaidGroups = append(rgs[writeCacheRG].newRaidGroups, newRg)
				break
			}
		}
		// Old raid group should exist on new pool spec changes
		if !isRaidGroupExist {
			return nil, errors.Errorf("removing raid group from pool spec is invalid operation")
		}
	}
	return rgs, nil
}

// ArePoolSpecChangesValid validates the pool specs on CSPC for raid groups
// changes(day-2-operations). Steps performed in this function
// 1. Get common raidgroups with index matching from old and new spec.
// 2. Iterate over common old and new raid groups and perform following steps:
//    2.1 Validate raid group changes.
//        2.1.1: Verify and return error when new block device added or removed from existing
//               raid groups for other than stripe pool type.
//    2.2 Validate changes for blockdevice replacement scenarios(openebs/openebs#2846).
// 3. Validate vertical pool expansions if there are any new raidgroups or blockdevices added.
func (pOps *PoolOperations) ArePoolSpecChangesValid(oldPoolSpec, newPoolSpec *cstor.PoolSpec) (bool, string) {
	if oldPoolSpec.PoolConfig.DataRaidGroupType != newPoolSpec.PoolConfig.DataRaidGroupType ||
		(oldPoolSpec.PoolConfig.WriteCacheGroupType != "" &&
			oldPoolSpec.PoolConfig.WriteCacheGroupType != newPoolSpec.PoolConfig.WriteCacheGroupType) {
		return false, fmt.Sprintf("raidgroup can't be modified")
	}
	newToOldBd := make(map[string]string)
	commonRaidGroups, err := getIndexedCommonRaidGroups(oldPoolSpec, newPoolSpec)
	if err != nil {
		return false, fmt.Sprintf("raid group validation failed: %v", err)
	}
	for _, v := range commonRaidGroups {
		rgType := v.rgType
		for index, _ := range v.oldRaidGroups {
			oldRg := v.oldRaidGroups[index]
			// Already mapped(via index) old raid groups and new raid groups in
			// commonRaidGroups no need to iterate over v.newRaidGroups
			newRg := v.newRaidGroups[index]

			if err = validateRaidGroupChanges(&oldRg, &newRg, rgType); err != nil {
				return false, fmt.Sprintf("raid group validation failed: %v", err)
			}
			if IsBlockDeviceReplacementCase(&oldRg, &newRg) {
				if ok, msg := pOps.IsBDReplacementValid(&newRg, &oldRg, rgType); !ok {
					return false, msg
				}
				newBD := GetNewBDFromRaidGroups(&newRg, &oldRg)
				for k, v := range newBD {
					newToOldBd[k] = v
				}
			}
		}
	}

	err = pOps.validatePoolExpansion(newPoolSpec, commonRaidGroups)
	if err != nil {
		return false, fmt.Sprintf("pool expansion validation failed: %v", err)
	}

	for newBD, oldBD := range newToOldBd {
		err := pOps.createBDC(newBD, oldBD)
		if err != nil {
			return false, err.Error()
		}
	}
	return true, ""
}
