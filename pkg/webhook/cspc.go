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
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"reflect"
	"strings"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	"github.com/openebs/api/v3/pkg/apis/types"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
)

// TODO: Make better naming conventions from review comments

// PoolValidator is build to validate pool spec, raid groups and blockdevices
type PoolValidator struct {
	poolSpec  *cstor.PoolSpec
	namespace string
	hostName  string
	cspcName  string
	clientset clientset.Interface
}

type getCSPC func(name, namespace string, clientset clientset.Interface) (*cstor.CStorPoolCluster, error)

func getCSPCObject(name, namespace string,
	clientset clientset.Interface) (*cstor.CStorPoolCluster, error) {
	return clientset.CstorV1().
		CStorPoolClusters(namespace).
		Get(context.TODO(), name, metav1.GetOptions{})
}

// Builder is the builder object for Builder
type Builder struct {
	object *PoolValidator
}

// NewPoolSpecValidator returns new instance of poolValidator
func NewPoolSpecValidator() *PoolValidator {
	return &PoolValidator{}
}

// NewBuilder returns new instance of builder
func NewBuilder() *Builder {
	return &Builder{object: NewPoolSpecValidator()}
}

// build returns built instance of PoolValidator
func (b *Builder) build() *PoolValidator {
	return b.object
}

// withPoolSpec sets the poolSpec field of PoolValidator with provided values
func (b *Builder) withPoolSpec(poolSpec cstor.PoolSpec) *Builder {
	b.object.poolSpec = &poolSpec
	return b
}

// withPoolNamespace sets the namespace field of poolValidator with provided
// values
func (b *Builder) withPoolNamespace() *Builder {
	b.object.namespace = os.Getenv(util.OpenEBSNamespace)
	return b
}

// withClientset sets the clientset field of poolValidator with provided
// values
func (b *Builder) withClientset(c clientset.Interface) *Builder {
	b.object.clientset = c
	return b
}

// withPoolNodeName sets the node name field of poolValidator with provided
// values
func (b *Builder) withPoolNodeName(nodeName string) *Builder {
	b.object.hostName = nodeName
	return b
}

// withCSPCName sets the cspc name field of poolValidator with provided argument
func (b *Builder) withCSPCName(cspcName string) *Builder {
	b.object.cspcName = cspcName
	return b
}

// validateCSPC validates CSPC spec for Create, Update and Delete operation of the object.
func (wh *webhook) validateCSPC(ar *v1.AdmissionReview) *v1.AdmissionResponse {
	req := ar.Request
	response := &v1.AdmissionResponse{}
	// validates only if requested operation is CREATE or UPDATE
	if req.Operation == v1.Update {
		klog.V(5).Infof("Admission webhook update request for type %s", req.Kind.Kind)
		return wh.validateCSPCUpdateRequest(req, getCSPCObject)
	} else if req.Operation == v1.Create {
		klog.V(5).Infof("Admission webhook create request for type %s", req.Kind.Kind)
		return wh.validateCSPCCreateRequest(req)
	} else if req.Operation == v1.Delete {
		klog.V(5).Infof("Admission webhook delete request for type %s", req.Kind.Kind)
		return wh.validateCSPCDeleteRequest(req)
	}

	return response
}

// validateCSPCCreateRequest validates CSPC create request
func (wh *webhook) validateCSPCCreateRequest(req *v1.AdmissionRequest) *v1.AdmissionResponse {
	response := NewAdmissionResponse().SetAllowed().WithResultAsSuccess(http.StatusAccepted).AR
	var cspc cstor.CStorPoolCluster
	err := json.Unmarshal(req.Object.Raw, &cspc)
	if err != nil {
		klog.Errorf("Could not unmarshal cspc %s raw object: %v, %v", req.Name, err, req.Object.Raw)
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusBadRequest).AR
		return response
	}
	if ok, msg := wh.cspcValidation(&cspc); !ok {
		err := errors.Errorf("invalid cspc specification: %s", msg)
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusUnprocessableEntity).AR
		return response
	}
	return response
}

// validateCSPCDeleteRequest validates CSPC delete request
// if any cvrs exist on the cspc pools then deletion is invalid
func (wh *webhook) validateCSPCDeleteRequest(req *v1.AdmissionRequest) *v1.AdmissionResponse {
	response := NewAdmissionResponse().SetAllowed().WithResultAsSuccess(http.StatusAccepted).AR
	cspiList, err := wh.clientset.CstorV1().CStorPoolInstances(req.Namespace).List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: types.CStorPoolClusterLabelKey + "=" + req.Name,
		})
	if err != nil {
		klog.Errorf("Could not list cspi for cspc %s: %s", req.Name, err.Error())
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusBadRequest).AR
		return response
	}
	for _, cspiObj := range cspiList.Items {
		// list cvrs in all namespaces
		cvrList, err := wh.clientset.CstorV1().CStorVolumeReplicas("").List(context.TODO(), metav1.ListOptions{
			LabelSelector: "cstorpoolinstance.openebs.io/name=" + cspiObj.Name,
		})
		if err != nil {
			klog.Errorf("Could not list cvr for cspi %s: %s", cspiObj.Name, err.Error())
			response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusBadRequest).AR
			return response
		}
		if len(cvrList.Items) != 0 {
			err := errors.Errorf("invalid cspc %s deletion: volume still exists on pool %s", req.Name, cspiObj.Name)
			response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusUnprocessableEntity).AR
			return response
		}
	}
	return response
}

func (wh *webhook) cspcValidation(cspc *cstor.CStorPoolCluster) (bool, string) {
	usedNodes := map[string]bool{}
	if len(cspc.Spec.Pools) == 0 {
		return false, fmt.Sprintf("pools in cspc should have at least one item")
	}

	repeatedBlockDevices := getDuplicateBlockDeviceList(cspc)
	if len(repeatedBlockDevices) > 0 {
		return false, fmt.Sprintf("invalid cspc: cspc {%s} has duplicate blockdevices entries %v",
			cspc.Name,
			repeatedBlockDevices)
	}

	buildPoolValidator := NewBuilder().
		withPoolNamespace().
		withCSPCName(cspc.Name).
		withClientset(wh.clientset)
	for _, pool := range cspc.Spec.Pools {
		pool := pool // pin it
		nodeName, err := GetHostNameFromLabelSelector(pool.NodeSelector, wh.kubeClient)
		if err != nil {
			return false, fmt.Sprintf(
				"failed to get node from pool nodeSelector: {%v} error: {%v}",
				pool.NodeSelector,
				err,
			)
		}
		if usedNodes[nodeName] {
			return false, fmt.Sprintf("invalid cspc: duplicate node %s entry", nodeName)
		}
		usedNodes[nodeName] = true
		pValidate := buildPoolValidator.withPoolSpec(pool).
			withPoolNamespace().
			withPoolNodeName(nodeName).build()
		ok, msg := pValidate.poolSpecValidation()
		if !ok {
			return false, fmt.Sprintf("invalid pool spec: %s", msg)
		}
	}
	return true, ""
}

// getDuplicateBlockDeviceList returns list of block devices that are
// duplicated in CSPC
func getDuplicateBlockDeviceList(cspc *cstor.CStorPoolCluster) []string {
	duplicateBlockDeviceList := []string{}
	blockDeviceMap := map[string]bool{}
	addedBlockDevices := map[string]bool{}
	for _, poolSpec := range cspc.Spec.Pools {
		rgs := append(poolSpec.DataRaidGroups, poolSpec.WriteCacheRaidGroups...)
		for _, raidGroup := range rgs {
			for _, bd := range raidGroup.CStorPoolInstanceBlockDevices {
				// update duplicateBlockDeviceList only if block device is
				// repeated in CSPC and doesn't exist in duplicate block device
				// list.
				if blockDeviceMap[bd.BlockDeviceName] &&
					!addedBlockDevices[bd.BlockDeviceName] {
					duplicateBlockDeviceList = append(
						duplicateBlockDeviceList,
						bd.BlockDeviceName)
					addedBlockDevices[bd.BlockDeviceName] = true
				} else if !blockDeviceMap[bd.BlockDeviceName] {
					blockDeviceMap[bd.BlockDeviceName] = true
				}
			}
		}
	}
	return duplicateBlockDeviceList
}

func (poolValidator *PoolValidator) poolSpecValidation() (bool, string) {
	// TODO : Add validation for pool config
	// Pool config will require mutating webhooks also.
	ok, msg := poolValidator.poolConfigValidation(poolValidator.poolSpec.PoolConfig)
	if !ok {
		return false, msg
	}
	if len(poolValidator.poolSpec.DataRaidGroups) == 0 {
		return false, "at least one raid group should be present on dataRaidGroups"
	}
	if poolValidator.poolSpec.PoolConfig.DataRaidGroupType == string(cstor.PoolStriped) &&
		len(poolValidator.poolSpec.DataRaidGroups) != 1 {
		return false, "stripe dataRaidGroups should have exactly one raidGroup"
	}
	for _, raidGroup := range poolValidator.poolSpec.DataRaidGroups {
		raidGroup := raidGroup // pin it
		ok, msg := poolValidator.raidGroupValidation(&raidGroup, poolValidator.poolSpec.PoolConfig.DataRaidGroupType)
		if !ok {
			return false, msg
		}
	}
	// if raid groups are mentioned for writecache then validate
	if len(poolValidator.poolSpec.WriteCacheRaidGroups) != 0 {
		if poolValidator.poolSpec.PoolConfig.WriteCacheGroupType == string(cstor.PoolStriped) &&
			len(poolValidator.poolSpec.WriteCacheRaidGroups) != 1 {
			return false, "stripe writeCacheRaidGroups should have exactly one raidGroup"
		}
		for _, raidGroup := range poolValidator.poolSpec.WriteCacheRaidGroups {
			raidGroup := raidGroup // pin it
			ok, msg := poolValidator.raidGroupValidation(&raidGroup, poolValidator.poolSpec.PoolConfig.WriteCacheGroupType)
			if !ok {
				return false, msg
			}
		}
	}

	return true, ""
}

type validateRaidBDCount func(int) (bool, string)

func isStripedBDCountValid(count int) (bool, string) {
	if count < 1 {
		return false, fmt.Sprint("stripe raid group should have atleast one disk")
	}
	return true, ""
}

func isMirroredBDCountValid(count int) (bool, string) {
	if count%2 != 0 {
		return false, fmt.Sprint("mirror raid group should have disks in multiple of 2")
	}
	return true, ""
}

func isRaidzBDCountValid(count int) (bool, string) {
	// the number of disk should be 2^n+1 where n >= 1
	n := count - 1
	x := math.Ceil(math.Log2(float64(n)))
	y := math.Floor(math.Log2(float64(n)))
	if x != y || x < 1 {
		return false, fmt.Sprint("raidz raid group should have disks of the order 2^n+1, where n>0")
	}
	return true, ""
}

func isRaidz2BDCountValid(count int) (bool, string) {
	// the number of disk should be 2^n+2 where n >= 2
	n := count - 2
	x := math.Ceil(math.Log2(float64(n)))
	y := math.Floor(math.Log2(float64(n)))
	if x != y || x < 2 {
		return false, fmt.Sprint("raidz raid group should have disks of the order 2^n+2, where n>1")
	}
	return true, ""
}

var (
	// SupportedPRaidType is a map holding the supported raid configurations
	// Value of the keys --
	// 1. In case of striped this is the minimum number of disk required.
	// 2. In all other cases this is the exact number of disks required.
	SupportedPRaidType = map[cstor.PoolType]validateRaidBDCount{
		cstor.PoolStriped:  isStripedBDCountValid,
		cstor.PoolMirrored: isMirroredBDCountValid,
		cstor.PoolRaidz:    isRaidzBDCountValid,
		cstor.PoolRaidz2:   isRaidz2BDCountValid,
	}
	// SupportedCompression is a map holding the supported compressions
	// TODO: support to allow gzip-[1-9] compression algorithm also can be added later if requested
	// by the community.
	SupportedCompression = map[string]bool{
		"on":   true,
		"off":  true,
		"lzjb": true,
		"gzip": true,
		"zle":  true,
		"":     true,
		"lz4":  true,
	}
)

func (poolValidator *PoolValidator) poolConfigValidation(
	poolConfig cstor.PoolConfig) (bool, string) {
	if poolConfig.DataRaidGroupType == "" {
		return false, fmt.Sprintf("missing dataRaidGroupType")
	}
	if _, ok := SupportedPRaidType[cstor.PoolType(poolConfig.DataRaidGroupType)]; !ok {
		return false, fmt.Sprintf("unsupported dataRaidGroupType '%s' specified", poolConfig.DataRaidGroupType)
	}
	if _, ok := SupportedCompression[poolConfig.Compression]; !ok {
		return false, fmt.Sprintf("unsupported compression '%s' specified", poolConfig.Compression)
	}
	// if raid groups are mentioned for writecache then validate
	if len(poolValidator.poolSpec.WriteCacheRaidGroups) != 0 {
		if poolConfig.WriteCacheGroupType == "" {
			return false, fmt.Sprintf("missing writeCacheGroupType")
		}
		if _, ok := SupportedPRaidType[cstor.PoolType(poolConfig.WriteCacheGroupType)]; !ok {
			return false, fmt.Sprintf("unsupported writeCacheGroupType '%s' specified", poolConfig.WriteCacheGroupType)
		}
	}
	return true, ""
}

func (poolValidator *PoolValidator) raidGroupValidation(
	raidGroup *cstor.RaidGroup, rgType string) (bool, string) {

	if len(raidGroup.CStorPoolInstanceBlockDevices) == 0 {
		return false, fmt.Sprintf("empty raid group: number of block devices honouring raid type should be specified")
	}

	if ok, msg := SupportedPRaidType[cstor.PoolType(rgType)](len(raidGroup.CStorPoolInstanceBlockDevices)); !ok {
		return false, msg
	}

	for _, bd := range raidGroup.CStorPoolInstanceBlockDevices {
		bd := bd
		ok, msg := poolValidator.blockDeviceValidation(&bd)
		if !ok {
			return false, msg
		}
	}
	return true, ""
}

func validateBlockDevice(bd *openebsapis.BlockDevice, hostName string) error {
	if bd.Status.State != "Active" {
		return errors.Errorf(
			"block device %q is in not in active state", bd.Name,
		)
	}
	// In case of migration from SPC to CSPC there might be chances of blockdevice
	// CR having FSType as "zfs_member" if devices are consumed by cStor. This FSType
	// will present only if cStor is created on top of partitioned blockdevice.
	// TODO: Check type of FS if blockdevice is consuming by ZFS LOCALPV. This case
	// will be covered when we will check for blockdevice tag
	if bd.Spec.FileSystem.Type != "" && bd.Spec.FileSystem.Type != "zfs_member" {
		return errors.Errorf("block device has file system {%s}",
			bd.Spec.FileSystem.Type,
		)
	}
	if bd.Labels[types.HostNameLabelKey] != hostName {
		return errors.Errorf(
			"block device %s doesn't belongs to node %s",
			bd.Name,
			bd.Labels[types.HostNameLabelKey],
		)
	}

	// If the BD tag is present on BD and the value is empty then
	// this BD is not a valid BD for provisioning.
	if v, found := bd.Labels[types.BlockDeviceTagLabelKey]; found {
		if strings.TrimSpace(v) == "" {
			return errors.Errorf(
				"block device %s is tagged with a value %s and cannot be used",
				bd.Name,
				v,
			)
		}
	}

	return nil
}

// blockDeviceValidation validates following steps:
// 1. block device name shouldn't be empty.
// 2. If block device has claim it verifies whether claim is created by this CSPC
func (poolValidator *PoolValidator) blockDeviceValidation(
	bd *cstor.CStorPoolInstanceBlockDevice) (bool, string) {
	if bd.BlockDeviceName == "" {
		return false, fmt.Sprint("block device name cannot be empty")
	}
	bdObj, err := poolValidator.clientset.OpenebsV1alpha1().BlockDevices(poolValidator.namespace).
		Get(context.TODO(), bd.BlockDeviceName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Sprintf(
			"failed to get block device: {%s} details error: %v",
			bd.BlockDeviceName,
			err,
		)
	}
	err = validateBlockDevice(bdObj, poolValidator.hostName)

	if err != nil {
		return false, fmt.Sprintf("%v", err)
	}
	if bdObj.Status.ClaimState == openebsapis.BlockDeviceClaimed {
		// TODO: Need to check how NDM
		if bdObj.Spec.ClaimRef != nil {
			bdcName := bdObj.Spec.ClaimRef.Name
			if err := poolValidator.blockDeviceClaimValidation(bdcName, bdObj.Name); err != nil {
				return false, fmt.Sprintf("error: %v", err)
			}
		}
	}
	return true, ""
}

func (poolValidator *PoolValidator) blockDeviceClaimValidation(bdcName, bdName string) error {
	bdcObject, err := poolValidator.clientset.OpenebsV1alpha1().BlockDeviceClaims(poolValidator.namespace).
		Get(context.TODO(), bdcName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err,
			"could not get block device claim for block device {%s}", bdName)
	}
	cspcName := bdcObject.
		GetLabels()[types.CStorPoolClusterLabelKey]
	if cspcName != poolValidator.cspcName {
		return errors.Errorf("can't use claimed blockdevice %s", bdName)
	}
	return nil
}

// validateCSPCUpdateRequest validates CSPC update request
// ToDo: Remove repetitive code.
func (wh *webhook) validateCSPCUpdateRequest(req *v1.AdmissionRequest, getCSPC getCSPC) *v1.AdmissionResponse {
	response := NewAdmissionResponse().SetAllowed().WithResultAsSuccess(http.StatusAccepted).AR
	var cspcNew cstor.CStorPoolCluster
	err := json.Unmarshal(req.Object.Raw, &cspcNew)
	if err != nil {
		klog.Errorf("Could not unmarshal cspc %s raw object: %v, %+v", req.Name, err, string(req.Object.Raw))
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusBadRequest).AR
		return response
	}
	// Get CSPC old object
	cspcOld, err := getCSPC(cspcNew.Name, cspcNew.Namespace, wh.clientset)
	if err != nil {
		err = errors.Errorf("could not fetch existing cspc for validation: %s", err.Error())
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusInternalServerError).AR
		return response
	}

	// return success from here when there is no change in old and new spec
	if reflect.DeepEqual(cspcNew.Spec, cspcOld.Spec) {
		return response
	}
	if ok, msg := wh.cspcValidation(&cspcNew); !ok {
		err = errors.Errorf("invalid cspc specification: %s", msg)
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusUnprocessableEntity).AR
		return response
	}
	pOps := NewPoolOperations(wh.kubeClient, wh.clientset).WithNewCSPC(&cspcNew).WithOldCSPC(cspcOld)

	if ok, msg := pOps.ValidateScaledown(); !ok {
		err = errors.Errorf("invalid cspc specification: %s", msg)
		// As scale down validation may take more time than the timeout value set
		// on webhook having a log will help debug
		klog.Error(err)
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusUnprocessableEntity).AR
		return response
	}

	commonPoolSpec, err := getCommonPoolSpecs(&cspcNew, cspcOld, wh.kubeClient)

	if err != nil {
		err = errors.Errorf("could not find common pool specs for validation: %s", err.Error())
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusInternalServerError).AR
		return response
	}
	if ok, msg := ValidateSpecChanges(commonPoolSpec, pOps); !ok {
		err = errors.Errorf("invalid cspc specification: %s", msg)
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusUnprocessableEntity).AR
		return response
	}

	return response
}

// ValidateScaledown validates whether any cvr exist on the cspi
// that is being scaled down
func (p *PoolOperations) ValidateScaledown() (bool, string) {
	removedPools := []string{}

	for _, oldPool := range p.OldCSPC.Spec.Pools {
		found := false
		for _, newPool := range p.NewCSPC.Spec.Pools {
			if reflect.DeepEqual(oldPool.NodeSelector, newPool.NodeSelector) {
				found = true
				break
			}
		}
		if !found && p.IsScaledownCase(oldPool) {
			var nodeName string
			if v, ok := oldPool.NodeSelector[types.HostNameLabelKey]; ok {
				nodeName = v
			} else {
				gotNodeName, err := GetHostNameFromLabelSelector(oldPool.NodeSelector, p.kubeClient)
				if err != nil {
					return false, fmt.Sprintf("Could not list node for node selectors {%v}", oldPool.NodeSelector)
				}
				nodeName = gotNodeName
			}

			ls := &metav1.LabelSelector{
				MatchLabels: map[string]string{
					types.CStorPoolClusterLabelKey: p.OldCSPC.Name,
					types.HostNameLabelKey:         nodeName,
				},
			}
			cspi, err := p.clientset.CstorV1().CStorPoolInstances(p.OldCSPC.Namespace).
				List(context.TODO(), metav1.ListOptions{
					LabelSelector: labels.Set(ls.MatchLabels).String(),
				})
			if err != nil {
				return false, fmt.Sprintf("Could not list cspi for cspc %s: %s", p.OldCSPC.Name, err.Error())
			}
			removedPools = append(removedPools, cspi.Items[0].Name)
		}
	}
	for _, cspiName := range removedPools {
		// list cvrs in cspc namespaces
		cvrList, err := p.clientset.CstorV1().CStorVolumeReplicas(p.OldCSPC.Namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: types.CStorPoolInstanceNameLabelKey + "=" + cspiName,
		})
		if err != nil {
			return false, fmt.Sprintf("Could not list cvr for cspi %s: %s", cspiName, err.Error())
		}
		if len(cvrList.Items) != 0 {
			return false, fmt.Sprintf("volume still exists on pool %s", cspiName)
		}
	}
	return true, ""
}

// IsScaledownCase checks whether it is scale down case or the node selector
// for an exsiting pool got changed.
func (p *PoolOperations) IsScaledownCase(oldPool cstor.PoolSpec) bool {
	bdMap := map[string]int{}
	for _, newPool := range p.NewCSPC.Spec.Pools {
		for _, newRg := range append(newPool.DataRaidGroups, newPool.WriteCacheRaidGroups...) {
			for _, newBD := range newRg.CStorPoolInstanceBlockDevices {
				bdMap[newBD.BlockDeviceName]++
			}
		}
	}
	for _, oldRg := range append(oldPool.DataRaidGroups, oldPool.WriteCacheRaidGroups...) {
		for _, oldBD := range oldRg.CStorPoolInstanceBlockDevices {
			if bdMap[oldBD.BlockDeviceName] > 0 {
				return false
			}
		}
	}
	return true
}
