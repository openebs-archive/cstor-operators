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

package cstorvolumeconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	version "github.com/hashicorp/go-version"
	cstorapis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	cstortypes "github.com/openebs/api/v3/pkg/apis/types"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	cstorversion "github.com/openebs/cstor-operators/pkg/version"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// restoreAPIOps required to perform CRUD operations
// on restore
type restoreAPIOps struct {
	req          *http.Request
	resp         http.ResponseWriter
	k8sclientset kubernetes.Interface
	clientset    clientset.Interface
	namespace    string
}

// restoreV1alpha1SpecificRequest deals with restore API requests
func (s *HTTPServer) restoreV1alpha1SpecificRequest(
	resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	restoreOps := &restoreAPIOps{
		req:          req,
		resp:         resp,
		k8sclientset: s.cvcServer.kubeclientset,
		clientset:    s.cvcServer.clientset,
		namespace:    getOpenEBSNamespace(),
	}

	switch req.Method {
	case "POST":
		klog.Infof("Got restore Create request")
		return restoreOps.create()
	case "GET":
		klog.Infof("Got restore GET request")
		return restoreOps.get()
	}
	klog.Infof("restore endpoint doesn't support %s", req.Method)
	return nil, CodedError(405, ErrInvalidMethod)
}

// Create is http handler which handles restore-create request
func (rOps *restoreAPIOps) create() (interface{}, error) {
	var err error
	restore := &cstorapis.CStorRestore{}
	err = decodeBody(rOps.req, restore)
	if err != nil {
		return nil, err
	}

	err = validateCreateRestoreRequest(restore)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("restore create validation failed error: {%s}", err.Error()))
	}

	err = rOps.createVolumeForRestore(restore)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to create resources for volume: {%v}", err))
	}
	klog.Infof("Restore volume '%v' created successfully ", restore.Spec.VolumeName)

	if restore.Spec.Local {
		return getISCSIPersistentVolumeSource(restore.Spec.VolumeName, rOps.namespace, rOps.clientset)
	}

	return createRestore(rOps.clientset, restore)
}

// get is http handler which handles restore get request
func (rOps *restoreAPIOps) get() (interface{}, error) {
	var resp []byte

	rst := &cstorapis.CStorRestore{}

	err := decodeBody(rOps.req, rst)
	if err != nil {
		return nil, err
	}

	// restore name is expected
	if len(strings.TrimSpace(rst.Spec.RestoreName)) == 0 {
		return nil, CodedError(400, fmt.Sprintf("Failed to get restore: missing restore name "))
	}

	// namespace is expected
	if len(strings.TrimSpace(rst.Namespace)) == 0 {
		return nil, CodedError(400, fmt.Sprintf("Failed to get restore '%v': missing namespace", rst.Spec.RestoreName))
	}

	// volume name is expected
	if len(strings.TrimSpace(rst.Spec.VolumeName)) == 0 {
		return nil, CodedError(400, fmt.Sprintf("Failed to get restore '%v': missing volume name", rst.Spec.RestoreName))
	}

	rstatus, err := rOps.getRestoreStatus(rst)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to fetch status '%v'", err))
	}

	resp, err = json.Marshal(rstatus)
	if err == nil {
		_, err = rOps.resp.Write(resp)
		if err != nil {
			return nil, CodedError(400, fmt.Sprintf("Failed to send response data"))
		}
		return nil, nil
	}

	return nil, CodedError(400, fmt.Sprintf("Failed to encode response data"))
}

// createVolumeForRestore creates CVC object only if it is local restore request
// else retun error if CVC status is not in Bound state else nil will be returned
func (rOps *restoreAPIOps) createVolumeForRestore(restoreObj *cstorapis.CStorRestore) error {

	// If the request is to restore local backup then velero-plugin will not create PVC.
	// So let's create CVC with annotation "openebs.io/created-through" which will be propagated
	// to CVRs. If CVR controller observe this annotation then it will not set targetIP.
	if restoreObj.Spec.Local {
		// 1. Fetch the storageclass from etcd
		// 2. Validate the storageclass whether it has required details to create CVC
		scObj, err := rOps.k8sclientset.StorageV1().StorageClasses().Get(context.TODO(), restoreObj.Spec.StorageClass, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to get storageclass %s", restoreObj.Spec.StorageClass)
		}
		err = validateStorageClassParameters(scObj)
		if err != nil {
			return errors.Wrapf(err, "Storageclass parametes validation failed")
		}
		// Build CStorVolumeConfig
		cvcObj, err := rOps.buildCStorVolumeConfig(scObj, restoreObj)
		if err != nil {
			return errors.Wrapf(err, "failed to build CVC to provision cstor volume")
		}
		_, err = rOps.clientset.CstorV1().CStorVolumeConfigs(rOps.namespace).Create(context.TODO(), cvcObj, metav1.CreateOptions{})
		if err != nil && !k8serror.IsAlreadyExists(err) {
			return errors.Wrapf(err, "failed to create CVC %s object", cvcObj.Name)
		}
		klog.Infof("successfully created cvc %s in namespace %s", cvcObj.Name, cvcObj.Namespace)
	}

	// In case of CStor CSI volumes, if CVC.Status.Phase is marked as Bound then
	// all the resources are created.
	return waitForCVCBoundState(restoreObj.Spec.VolumeName, rOps.namespace, rOps.clientset)
}

// getRestoreStatus returns the status of CStorRestore
func (rOps *restoreAPIOps) getRestoreStatus(rst *cstorapis.CStorRestore) (cstorapis.CStorRestoreStatus, error) {
	rstStatus := cstorapis.RSTCStorStatusEmpty

	listOptions := metav1.ListOptions{
		LabelSelector: "openebs.io/restore=" + rst.Spec.RestoreName + "," +
			cstortypes.PersistentVolumeLabelKey + "=" + rst.Spec.VolumeName,
	}
	// NOTE: CStorPoolInstances of same pool cluster may be in different versions

	// If v1alpha1 resource doesn't exist then listing resources will get NotFound error
	v1Alpha1RestoreList, err := rOps.clientset.OpenebsV1alpha1().CStorRestores(rst.Namespace).List(context.TODO(), listOptions)
	if err != nil && !k8serror.IsNotFound(err) {
		return cstorapis.RSTCStorStatusEmpty, CodedError(400, fmt.Sprintf("Failed to fetch restore error:%v", err))
	}

	v1RestoreList, err := rOps.clientset.CstorV1().CStorRestores(rst.Namespace).List(context.TODO(), listOptions)
	if err != nil && !k8serror.IsNotFound(err) {
		return cstorapis.RSTCStorStatusEmpty, CodedError(400, fmt.Sprintf("Failed to fetch restore error:%v", err))
	}

	// Get restore status of v1alpha1 CStorRestore objects
	if len(v1Alpha1RestoreList.Items) != 0 {
		rstStatus = cstorapis.CStorRestoreStatus(rOps.getV1Alpha1CStorRestoreStatus(v1Alpha1RestoreList))
		if rstStatus == cstorapis.RSTCStorStatusInProgress {
			return rstStatus, nil
		}
	}

	// Get restore status of v1 CStorRestore objects
	if len(v1RestoreList.Items) != 0 {
		rstStatus = rOps.getCStorRestoreStatus(v1RestoreList)
	}
	return rstStatus, nil
}

// createRestore create restore CR for volume's CVR
func createRestore(openebsClient clientset.Interface, restoreObj *cstorapis.CStorRestore) (interface{}, error) {
	// TODO: Need to check changes related to namespace
	namespace := getOpenEBSNamespace()
	//Get List of cvr's related to this pvc
	listOptions := metav1.ListOptions{
		LabelSelector: cstortypes.PersistentVolumeLabelKey + "=" + restoreObj.Spec.VolumeName,
	}
	cvrList, err := openebsClient.CstorV1().CStorVolumeReplicas("").List(context.TODO(), listOptions)
	if err != nil {
		return nil, CodedError(500, err.Error())
	}
	if len(cvrList.Items) == 0 {
		return nil, CodedError(500, fmt.Sprintf("CVRs doesn't exist for volume %s", restoreObj.Spec.VolumeName))
	}
	// Below flow:
	// 1. Get CSPC and list all the CSPIs name with that CSPC.
	// 2. Select CSPIs which has restore CVRs of this particular volume.
	// 3. Get the pool version details from above selected CSPIs.
	// 4. Based on the version of CSPI create supported version of restore objects.
	cspiName := cvrList.Items[0].Labels[cstortypes.CStorPoolInstanceNameLabelKey]
	lastIndex := strings.LastIndex(cspiName, "-")
	cspcName := cspiName[:lastIndex]
	cspiList, err := openebsClient.CstorV1().
		CStorPoolInstances("").
		List(context.TODO(), metav1.ListOptions{
			LabelSelector: cstortypes.CStorPoolClusterLabelKey + "=" + cspcName,
		})
	if err != nil {
		return nil, CodedError(500, fmt.Sprintf("failed to list CSPIs of CSPC %s error: %v", cspcName, err))
	}

	cspiToCVRMap := make(map[string]string, len(cvrList.Items))
	for _, cvr := range cvrList.Items {
		cspiToCVRMap[cvr.Labels[cstortypes.CStorPoolInstanceNameLabelKey]] = cvr.Name
	}

	poolVersionMap := make(map[string]*version.Version, len(cvrList.Items))
	for _, cspi := range cspiList.Items {
		cspiVersion := cspi.VersionDetails.Status.Current
		// Ignore pool which doesn't have restore CVRs
		if _, ok := cspiToCVRMap[cspi.Name]; !ok {
			continue
		}
		poolVersion, err := version.NewVersion(strings.Split(cspiVersion, "-")[0])
		// If Current version is empty treat it as a ci
		if err != nil && (cspiVersion != "" && !strings.Contains(cspiVersion, "dev")) {
			return nil, CodedError(500,
				fmt.Sprintf("failed to parse pool(%s) version %s error: %v", cspi.Name, cspi.VersionDetails.Status.Current, err))
		}
		poolVersionMap[cspi.Name] = poolVersion
	}

	v1Alpha1RestoreObj := getV1Alpha1RestoreFromV1(restoreObj)
	for _, cvr := range cvrList.Items {
		cspiName := cvr.GetLabels()[cstortypes.CStorPoolInstanceNameLabelKey]
		poolVersion := poolVersionMap[cspiName]

		if poolVersion != nil && poolVersion.LessThan(minV1SupportedVersion) {
			err = buildAndCreateV1Alpha1Restore(openebsClient, v1Alpha1RestoreObj, cvr)
		} else {
			err = buildAndCreateV1Restore(openebsClient, restoreObj, cvr)
		}
		if err != nil {
			return nil, CodedError(500, fmt.Sprintf("failed to create restore for %s error: %v", restoreObj.Spec.VolumeName, err))
		}
	}
	return getISCSIPersistentVolumeSource(restoreObj.Spec.VolumeName, namespace, openebsClient)
}

func getV1Alpha1RestoreFromV1(restore *cstorapis.CStorRestore) *openebsapis.CStorRestore {
	return &openebsapis.CStorRestore{
		ObjectMeta: restore.ObjectMeta,
		Spec: openebsapis.CStorRestoreSpec{
			RestoreName:   restore.Spec.RestoreName,
			VolumeName:    restore.Spec.VolumeName,
			RestoreSrc:    restore.Spec.RestoreSrc,
			MaxRetryCount: restore.Spec.MaxRetryCount,
			RetryCount:    restore.Spec.RetryCount,
			StorageClass:  restore.Spec.StorageClass,
			Size:          restore.Spec.Size,
			Local:         restore.Spec.Local,
		},
		Status: openebsapis.CStorRestoreStatus(restore.Status),
	}
}

func waitForCVCBoundState(cvcName, namespace string, clientset clientset.Interface) error {
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		cvcObj, err := clientset.CstorV1().CStorVolumeConfigs(namespace).Get(context.TODO(), cvcName, metav1.GetOptions{})
		// If CVC is not found then wait for it to exist in etcd
		if err != nil && !k8serror.IsNotFound(err) {
			return err
		}
		if cvcObj.Status.Phase == cstorapis.CStorVolumeConfigPhaseBound {
			// Which means all the CStorVolume related resources are created
			return nil
		}
		klog.Errorf("waiting for CVC: %s in namespace: %s to become Bounded error: %v", cvcObj.Name, cvcObj.Namespace, err)
		time.Sleep(2 * time.Second)
	}
	return errors.Errorf("CVC %s in namespace %s is not in Bound status", cvcName, namespace)
}

// ValidateCreateRestoreRequest will validate the restore creation request and
// returns error if any validation failed
func validateCreateRestoreRequest(restore *cstorapis.CStorRestore) error {
	// namespace is expected
	if !restore.Spec.Local && len(strings.TrimSpace(restore.Namespace)) == 0 {
		return errors.Errorf("failed to create restore '%v': missing namespace", restore.Name)
	}

	// restore name is expected
	if len(strings.TrimSpace(restore.Spec.RestoreName)) == 0 {
		return errors.Errorf("failed to create restore: missing restore name")
	}

	// volume name is expected
	if len(strings.TrimSpace(restore.Spec.VolumeName)) == 0 {
		return errors.Errorf("failed to create restore '%v': missing volume name", restore.Name)
	}

	// restoreIP is expected
	if len(strings.TrimSpace(restore.Spec.RestoreSrc)) == 0 {
		return errors.Errorf("failed to create restore '%v': missing restoreSrc", restore.Name)
	}

	// storageClass is expected if restore is for local snapshot
	if restore.Spec.Local && len(strings.TrimSpace(restore.Spec.StorageClass)) == 0 {
		return errors.Errorf("failed to create restore '%v': missing storageClass", restore.Name)
	}

	// size is expected if restore is for local snapshot
	if restore.Spec.Local && len(strings.TrimSpace(restore.Spec.Size.String())) == 0 {
		return errors.Errorf("failed to create restore '%v': missing size", restore.Name)
	}
	return nil
}

func validateStorageClassParameters(scObj *storagev1.StorageClass) error {
	if _, ok := scObj.Parameters["cstorPoolCluster"]; !ok {
		return errors.Errorf("storageclass %s doesn't have cstorPoolCluster details", scObj.Name)
	}
	if _, ok := scObj.Parameters["replicaCount"]; !ok {
		return errors.Errorf("storageclass %s doesn't have replica count", scObj.Name)
	}
	if _, err := strconv.Atoi(scObj.Parameters["replicaCount"]); err != nil {
		return errors.Wrapf(err, "failed to convert replica count %s into integer", scObj.Parameters["replicaCount"])
	}
	return nil
}

// buildCStorVolumeConfig build the CVC from StorageClass and restoreObject
// NOTE: This function will be called only in case of local restore request
func (rOps *restoreAPIOps) buildCStorVolumeConfig(
	scObj *storagev1.StorageClass, restoreObj *cstorapis.CStorRestore) (*cstorapis.CStorVolumeConfig, error) {
	replicaCount, _ := strconv.Atoi(scObj.Parameters["replicaCount"])

	// Build CStorVolumeConfig
	// TODO: Convert all literals into constants and make as builder code
	cvcObj := &cstorapis.CStorVolumeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreObj.Spec.VolumeName,
			Namespace: rOps.namespace,
			Labels: map[string]string{
				cstortypes.CStorPoolClusterLabelKey: scObj.Parameters["cstorPoolCluster"],
				"openebs.io/source-volume":          restoreObj.Spec.RestoreSrc,
			},
			Annotations: map[string]string{
				"openebs.io/volumeID":      restoreObj.Spec.VolumeName,
				"openebs.io/volume-policy": scObj.Parameters["cstorVolumePolicy"],
			},
			Finalizers: []string{"cvc.openebs.io/finalizer"},
		},
		Spec: cstorapis.CStorVolumeConfigSpec{
			Provision: cstorapis.VolumeProvision{
				Capacity: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): restoreObj.Spec.Size,
				},
				ReplicaCount: replicaCount,
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceName(corev1.ResourceStorage): restoreObj.Spec.Size,
			},
			// If CStorVolumeSource is mentioned then CVC controller treat it as a clone request
			// CStorVolumeSource contains the source volumeName@snapShotname
			CStorVolumeSource: restoreObj.Spec.RestoreSrc + "@" + restoreObj.Spec.RestoreName,
		},
		VersionDetails: cstorapis.VersionDetails{
			Status: cstorapis.VersionStatus{
				Current: cstorversion.Current(),
			},
			Desired: cstorversion.Current(),
		},
		Status: cstorapis.CStorVolumeConfigStatus{
			Phase: cstorapis.CStorVolumeConfigPhasePending,
		},
	}

	return cvcObj, nil
}

// getISCSIPersistentVolumeSource will return iscsipersistentvolumesource object
// by populating with the help of CStorVolume object
func getISCSIPersistentVolumeSource(
	volumeName, namespace string, clientset clientset.Interface) (interface{}, error) {
	cvObj, err := clientset.CstorV1().CStorVolumes(namespace).Get(context.TODO(), volumeName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get volume details %s in namespace %s", volumeName, namespace)
		return nil, err
	}
	iscsiPVSrc := &corev1.ISCSIPersistentVolumeSource{
		TargetPortal: cvObj.Spec.TargetPortal,
		IQN:          cvObj.Spec.Iqn,
		// Lun:          cvObj.Spec.Lun,
		// TODO: Need to check how can we get FSType without PVC
		// FSType:       cas.Spec.FSType,
		ReadOnly: false,
	}
	return iscsiPVSrc, nil
}

// isPoolManagerDown returns true if either pool manager pod/
// pool manager node went down
func isPoolManagerDown(k8sClient kubernetes.Interface,
	poolName, namespace string) bool {

	// check if node is running or not
	bkpNodeDown := checkIfPoolManagerNodeDown(k8sClient, poolName, namespace)
	// check if cstor-pool-mgmt container is running or not
	bkpPodDown := checkIfPoolManagerDown(k8sClient, poolName, namespace)

	if bkpNodeDown || bkpPodDown {
		return true
	}
	return false
}

func buildAndCreateV1Alpha1Restore(
	openebsClient clientset.Interface,
	restoreObj *openebsapis.CStorRestore,
	cvr cstorapis.CStorVolumeReplica,
) error {
	restoreObj.Name = restoreObj.Spec.RestoreName + "-" + string(uuid.NewUUID())
	_, err := openebsClient.
		OpenebsV1alpha1().
		CStorRestores(restoreObj.Namespace).
		Get(context.TODO(), restoreObj.Name, metav1.GetOptions{})
	if err != nil {
		restoreObj.Status = openebsapis.RSTCStorStatusPending
		restoreObj.ObjectMeta.Labels = map[string]string{
			cstortypes.CStorPoolInstanceNameLabelKey: cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey],
			cstortypes.CStorPoolInstanceUIDLabelKey:  cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceUIDLabelKey],
			cstortypes.PersistentVolumeLabelKey:      cvr.ObjectMeta.Labels[cstortypes.PersistentVolumeLabelKey],
			"openebs.io/restore":                     restoreObj.Spec.RestoreName,
		}

		_, err = openebsClient.OpenebsV1alpha1().CStorRestores(restoreObj.Namespace).Create(context.TODO(), restoreObj, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("Failed to create restore CR(volume:%s CSPI:%s) : error '%s'",
				restoreObj.Spec.VolumeName, cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey],
				err.Error())
			return err
		}
		klog.Infof("V1Alpha1 Restore:%s created for volume %q CSPI: %s", restoreObj.Name,
			restoreObj.Spec.VolumeName,
			restoreObj.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey])
	}
	return nil
}

func buildAndCreateV1Restore(
	openebsClient clientset.Interface,
	restoreObj *cstorapis.CStorRestore,
	cvr cstorapis.CStorVolumeReplica,
) error {
	cspiName := cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey]
	restoreObj.Name = restoreObj.Spec.RestoreName + "-" + string(uuid.NewUUID())
	_, err := openebsClient.
		CstorV1().
		CStorRestores(restoreObj.Namespace).
		Get(context.TODO(), restoreObj.Name, metav1.GetOptions{})
	if err != nil {
		restoreObj.Status = cstorapis.RSTCStorStatusPending
		restoreObj.ObjectMeta.Labels = map[string]string{
			cstortypes.CStorPoolInstanceNameLabelKey: cspiName,
			cstortypes.CStorPoolInstanceUIDLabelKey:  cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceUIDLabelKey],
			cstortypes.PersistentVolumeLabelKey:      cvr.ObjectMeta.Labels[cstortypes.PersistentVolumeLabelKey],
			"openebs.io/restore":                     restoreObj.Spec.RestoreName,
		}

		_, err = openebsClient.CstorV1().CStorRestores(restoreObj.Namespace).Create(context.TODO(), restoreObj, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("Failed to create restore CR(volume:%s CSPI:%s) : error '%s'",
				restoreObj.Spec.VolumeName, cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey],
				err.Error())
			return err
		}
		klog.Infof("Restore:%s created for volume %q CSPI: %s", restoreObj.Name,
			restoreObj.Spec.VolumeName,
			restoreObj.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey])
	}
	return nil
}

func (rOps *restoreAPIOps) getCStorRestoreStatus(
	restoreList *cstorapis.CStorRestoreList) cstorapis.CStorRestoreStatus {
	rstStatus := cstorapis.RSTCStorStatusEmpty
	namespace := getOpenEBSNamespace()

	for _, restore := range restoreList.Items {
		rstStatus = restore.Status
		if restore.Status != cstorapis.RSTCStorStatusDone &&
			restore.Status != cstorapis.RSTCStorStatusFailed {
			poolName := restore.Labels[cstortypes.CStorPoolInstanceNameLabelKey]
			isPoolDown := isPoolManagerDown(rOps.k8sclientset, poolName, namespace)
			if isPoolDown {
				rstStatus = cstorapis.RSTCStorStatusFailed
			}
		}

		switch rstStatus {
		case cstorapis.RSTCStorStatusInProgress:
			rstStatus = cstorapis.RSTCStorStatusInProgress
		case cstorapis.RSTCStorStatusFailed, cstorapis.RSTCStorStatusInvalid:
			if restore.Status != rstStatus {
				// Restore for given CVR may failed due to node failure or pool failure
				// Let's update status for given CVR's restore to failed
				restore.Status = rstStatus
				_, err := rOps.clientset.CstorV1().CStorRestores(restore.Namespace).Update(context.TODO(), &restore, metav1.UpdateOptions{})
				if err != nil {
					klog.Errorf("Failed to update restore:%s with status:%v", restore.Name, rstStatus)
				}
				rstStatus = cstorapis.RSTCStorStatusFailed
			}
		case cstorapis.RSTCStorStatusDone:
			if rstStatus != cstorapis.RSTCStorStatusFailed {
				rstStatus = cstorapis.RSTCStorStatusDone
			}
		}

		klog.Infof("Restore:%v status is %v", restore.Name, restore.Status)

		if rstStatus == cstorapis.RSTCStorStatusInProgress {
			break
		}
	}
	return rstStatus
}
