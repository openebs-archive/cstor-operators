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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/pkg/apis/openebs.io/v1alpha1"
	"github.com/openebs/api/pkg/apis/types"
	clientset "github.com/openebs/api/pkg/client/clientset/versioned"
	util "github.com/openebs/api/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// TODO: Make better naming conventions from review comments

// PoolValidator is build to validate pool spec, raid groups and blockdevices
type PoolValidator struct {
	poolSpec  *cstor.PoolSpec
	namespace string
	nodeName  string
	cspcName  string
	clientset clientset.Interface
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
	b.object.nodeName = nodeName
	return b
}

// withCSPCName sets the cspc name field of poolValidator with provided argument
func (b *Builder) withCSPCName(cspcName string) *Builder {
	b.object.cspcName = cspcName
	return b
}

// validateCSPC validates CSPC spec for Create, Update and Delete operation of the object.
func (wh *webhook) validateCSPC(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	response := &v1beta1.AdmissionResponse{}
	// validates only if requested operation is CREATE or UPDATE
	if req.Operation == v1beta1.Update {
		klog.V(5).Infof("Admission webhook update request for type %s", req.Kind.Kind)
		return wh.validateCSPCUpdateRequest(req)
	} else if req.Operation == v1beta1.Create {
		klog.V(5).Infof("Admission webhook create request for type %s", req.Kind.Kind)
		return wh.validateCSPCCreateRequest(req)
	} else if req.Operation == v1beta1.Delete {
		klog.V(5).Infof("Admission webhook delete request for type %s", req.Kind.Kind)
		return wh.validateCSPCDeleteRequest(req)
	}

	return response
}

// validateCSPCCreateRequest validates CSPC create request
func (wh *webhook) validateCSPCCreateRequest(req *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
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
func (wh *webhook) validateCSPCDeleteRequest(req *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	response := NewAdmissionResponse().SetAllowed().WithResultAsSuccess(http.StatusAccepted).AR
	cspiList, err := wh.clientset.CstorV1().CStorPoolInstances(req.Namespace).List(
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
		cvrList, err := wh.clientset.CstorV1().CStorVolumeReplicas("").List(metav1.ListOptions{
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
		for _, raidGroup := range poolSpec.DataRaidGroups {
			for _, bd := range raidGroup.BlockDevices {
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
	if len(poolValidator.poolSpec.DataRaidGroups) == 0 {
		return false, "at least one raid group should be present on pool spec"
	}
	// TODO : Add validation for pool config
	// Pool config will require mutating webhooks also.
	ok, msg := poolValidator.poolConfigValidation(poolValidator.poolSpec.PoolConfig)
	if !ok {
		return false, msg
	}
	for _, raidGroup := range poolValidator.poolSpec.DataRaidGroups {
		raidGroup := raidGroup // pin it
		ok, msg := poolValidator.raidGroupValidation(&raidGroup, poolValidator.poolSpec.PoolConfig.DataRaidGroupType)
		if !ok {
			return false, msg
		}
	}

	return true, ""
}

var (
	// SupportedPRaidType is a map holding the supported raid configurations
	// Value of the keys --
	// 1. In case of striped this is the minimum number of disk required.
	// 2. In all other cases this is the exact number of disks required.
	SupportedPRaidType = map[cstor.PoolType]int{
		cstor.PoolStriped:  1,
		cstor.PoolMirrored: 2,
		cstor.PoolRaidz:    3,
		cstor.PoolRaidz2:   6,
	}
)

func (poolValidator *PoolValidator) poolConfigValidation(
	poolConfig cstor.PoolConfig) (bool, string) {
	if poolConfig.DataRaidGroupType == "" {
		return false, fmt.Sprintf("missing dataRaidGroupType")
	}
	if _, ok := SupportedPRaidType[cstor.PoolType(poolConfig.DataRaidGroupType)]; !ok {
		return false, fmt.Sprintf("unsupported raid type '%s' specified", poolConfig.DataRaidGroupType)
	}
	return true, ""
}

func (poolValidator *PoolValidator) raidGroupValidation(
	raidGroup *cstor.RaidGroup, rgType string) (bool, string) {

	if len(raidGroup.BlockDevices) == 0 {
		return false, fmt.Sprintf("number of block devices honouring raid type should be specified")
	}

	if rgType != string(cstor.PoolStriped) {
		if len(raidGroup.BlockDevices) != SupportedPRaidType[cstor.PoolType(rgType)] {
			return false, fmt.Sprintf("number of block devices honouring raid type should be specified")
		}
	} else {
		if len(raidGroup.BlockDevices) < SupportedPRaidType[cstor.PoolType(rgType)] {
			return false, fmt.Sprintf("number of block devices honouring raid type should be specified")
		}
	}

	for _, bd := range raidGroup.BlockDevices {
		bd := bd
		ok, msg := poolValidator.blockDeviceValidation(&bd)
		if !ok {
			return false, msg
		}
	}
	return true, ""
}

func validateBlockDevice(bd *openebsapis.BlockDevice, nodeName string) error {
	if bd.Status.State != "Active" {
		return errors.Errorf(
			"block device is in not in active state",
		)
	}
	if bd.Spec.FileSystem.Type != "" {
		return errors.Errorf("block device has file system {%s}",
			bd.Spec.FileSystem.Type,
		)
	}
	if bd.Spec.NodeAttributes.NodeName != nodeName {
		return errors.Errorf(
			"block device %s doesn't belongs to node %s",
			bd.Name,
			bd.Spec.NodeAttributes.NodeName,
		)
	}
	return nil
}

// blockDeviceValidation validates following steps:
// 1. block device name shouldn't be empty.
// 2. If block device has claim it verifies whether claim is created by this CSPC
func (poolValidator *PoolValidator) blockDeviceValidation(
	bd *cstor.CStorPoolClusterBlockDevice) (bool, string) {
	if bd.BlockDeviceName == "" {
		return false, fmt.Sprint("block device name cannot be empty")
	}
	bdObj, err := poolValidator.clientset.OpenebsV1alpha1().BlockDevices(poolValidator.namespace).
		Get(bd.BlockDeviceName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Sprintf(
			"failed to get block device: {%s} details error: %v",
			bd.BlockDeviceName,
			err,
		)
	}
	err = validateBlockDevice(bdObj, poolValidator.nodeName)

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
		Get(bdcName, metav1.GetOptions{})
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
func (wh *webhook) validateCSPCUpdateRequest(req *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	response := NewAdmissionResponse().SetAllowed().WithResultAsSuccess(http.StatusAccepted).AR
	var cspcNew cstor.CStorPoolCluster
	err := json.Unmarshal(req.Object.Raw, &cspcNew)
	if err != nil {
		klog.Errorf("Could not unmarshal cspc %s raw object: %v, %v", req.Name, err, req.Object.Raw)
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusBadRequest).AR
		return response
	}
	// Get CSPC old object
	cspcOld, err := wh.clientset.CstorV1().CStorPoolClusters(cspcNew.Namespace).Get(cspcNew.Name, v1.GetOptions{})
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

	bdr := NewBlockDeviceReplacement(wh.kubeClient, wh.clientset).WithNewCSPC(&cspcNew).WithOldCSPC(cspcOld)
	commonPoolSpec, err := getCommonPoolSpecs(&cspcNew, cspcOld, wh.kubeClient)

	if err != nil {
		err = errors.Errorf("could not find common pool specs for validation: %s", err.Error())
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusInternalServerError).AR
		return response
	}

	if ok, msg := ValidateSpecChanges(commonPoolSpec, bdr); !ok {
		err = errors.Errorf("invalid cspc specification: %s", msg)
		response = BuildForAPIObject(response).UnSetAllowed().WithResultAsFailure(err, http.StatusUnprocessableEntity).AR
		return response
	}

	return response
}
