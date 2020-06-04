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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/pkg/apis/openebs.io/v1alpha1"
	cstortypes "github.com/openebs/api/pkg/apis/types"
	clientset "github.com/openebs/api/pkg/client/clientset/versioned"
	version "github.com/openebs/cstor-operators/pkg/version"
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
	restore := &openebsapis.CStorRestore{}
	err = decodeBody(rOps.req, restore)
	if err != nil {
		return nil, err
	}

	err = restoreCreateValidationRequest(restore)
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

	return createRestoreResource(rOps.clientset, restore)
}

// get is http handler which handles backup get request
func (rOps *restoreAPIOps) get() (interface{}, error) {
	var err error
	var rstatus openebsapis.CStorRestoreStatus
	var resp []byte

	rst := &openebsapis.CStorRestore{}

	err = decodeBody(rOps.req, rst)
	if err != nil {
		return nil, err
	}

	// backup name is expected
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

	rstatus, err = rOps.getRestoreStatus(rst)
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
// else it will retun error if CVC is not in Bound state else nil will be returned
func (rOps *restoreAPIOps) createVolumeForRestore(restoreObj *openebsapis.CStorRestore) error {

	// If the request is to restore local backup then velero-plugin will not create PVC.
	// So let's create CVC with annotation "openebs.io/created-through" which will be propagated
	// to CVRs. If CVR controller observe this annotation then it will not set targetIP.
	if restoreObj.Spec.Local {
		// 1. Fetch the storageclass from etcd
		// 2. Validate the storageclass whether it has required details to create CVC
		scObj, err := rOps.k8sclientset.StorageV1().StorageClasses().Get(restoreObj.Spec.StorageClass, metav1.GetOptions{})
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
		_, err = rOps.clientset.CstorV1().CStorVolumeConfigs(rOps.namespace).Create(cvcObj)
		if err != nil && !k8serror.IsAlreadyExists(err) {
			return errors.Wrapf(err, "failed to create CVC %s object", cvcObj.Name)
		}
		klog.Infof("successfully created cvc %s in namespace %s", cvcObj.Name, cvcObj.Namespace)
	}

	// In case of CStor CSI volumes, if CVC.Status.Phase is marked as Bound then
	// all the resources are created.
	err := waitForCVCBoundState(restoreObj.Spec.VolumeName, rOps.namespace, rOps.clientset)
	if err != nil {
		return err
	}
	return nil
}

// getRestoreStatus returns the status of CStorRestore
func (rOps *restoreAPIOps) getRestoreStatus(rst *openebsapis.CStorRestore) (openebsapis.CStorRestoreStatus, error) {
	rstStatus := openebsapis.RSTCStorStatusEmpty

	listOptions := metav1.ListOptions{
		LabelSelector: "openebs.io/restore=" + rst.Spec.RestoreName + "," +
			cstortypes.PersistentVolumeLabelKey + "=" + rst.Spec.VolumeName,
	}

	rlist, err := rOps.clientset.OpenebsV1alpha1().CStorRestores(rst.Namespace).List(listOptions)
	if err != nil {
		return openebsapis.RSTCStorStatusEmpty, CodedError(400, fmt.Sprintf("Failed to fetch restore error:%v", err))
	}

	for _, nr := range rlist.Items {
		rstStatus = getCStorRestoreStatus(rOps.k8sclientset, nr)

		switch rstStatus {
		case openebsapis.RSTCStorStatusInProgress:
			rstStatus = openebsapis.RSTCStorStatusInProgress
		case openebsapis.RSTCStorStatusFailed, openebsapis.RSTCStorStatusInvalid:
			if nr.Status != rstStatus {
				// Restore for given CVR may failed due to node failure or pool failure
				// Let's update status for given CVR's restore to failed
				updateRestoreStatus(rOps.clientset, nr, rstStatus)
			}
			rstStatus = openebsapis.RSTCStorStatusFailed
		case openebsapis.RSTCStorStatusDone:
			if rstStatus != openebsapis.RSTCStorStatusFailed {
				rstStatus = openebsapis.RSTCStorStatusDone
			}
		}

		klog.Infof("Restore:%v status is %v", nr.Name, nr.Status)

		if rstStatus == openebsapis.RSTCStorStatusInProgress {
			break
		}
	}
	return rstStatus, nil
}

// createRestoreResource create restore CR for volume's CVR
func createRestoreResource(openebsClient clientset.Interface, restoreObj *openebsapis.CStorRestore) (interface{}, error) {
	// TODO: Need to check changes related to namespace
	namespace := getOpenEBSNamespace()
	//Get List of cvr's related to this pvc
	listOptions := metav1.ListOptions{
		LabelSelector: cstortypes.PersistentVolumeLabelKey + "=" + restoreObj.Spec.VolumeName,
	}
	cvrList, err := openebsClient.CstorV1().CStorVolumeReplicas("").List(listOptions)
	if err != nil {
		return nil, CodedError(500, err.Error())
	}

	for _, cvr := range cvrList.Items {
		restoreObj.Name = restoreObj.Spec.RestoreName + "-" + string(uuid.NewUUID())
		oldrestoreObj, err := openebsClient.
			OpenebsV1alpha1().
			CStorRestores(restoreObj.Namespace).
			Get(restoreObj.Name, metav1.GetOptions{})
		if err != nil {
			restoreObj.Status = openebsapis.RSTCStorStatusPending
			restoreObj.ObjectMeta.Labels = map[string]string{
				cstortypes.CStorPoolInstanceNameLabelKey: cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey],
				cstortypes.CStorPoolInstanceUIDLabelKey:  cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceUIDLabelKey],
				cstortypes.PersistentVolumeLabelKey:      cvr.ObjectMeta.Labels[cstortypes.PersistentVolumeLabelKey],
				"openebs.io/restore":                     restoreObj.Spec.RestoreName,
			}

			_, err = openebsClient.OpenebsV1alpha1().CStorRestores(restoreObj.Namespace).Create(restoreObj)
			if err != nil {
				klog.Errorf("Failed to create restore CR(volume:%s CSPI:%s) : error '%s'",
					restoreObj.Spec.VolumeName, cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey],
					err.Error())
				return nil, CodedError(500, err.Error())
			}
			klog.Infof("Restore:%s created for volume %q CSPI: %s", restoreObj.Name,
				restoreObj.Spec.VolumeName,
				restoreObj.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey])
		} else {
			oldrestoreObj.Status = openebsapis.RSTCStorStatusPending
			oldrestoreObj.Spec = restoreObj.Spec
			_, err = openebsClient.OpenebsV1alpha1().CStorRestores(oldrestoreObj.Namespace).Update(oldrestoreObj)
			if err != nil {
				klog.Errorf("Failed to re-initialize old existing restore CR(volume:%s CSPI:%s) : error '%s'",
					restoreObj.Spec.VolumeName, cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey],
					err.Error())
				return nil, CodedError(500, err.Error())
			}
			klog.Infof("Re-initialized old restore:%s  %q CSPI:%v", restoreObj.Name,
				restoreObj.Spec.VolumeName,
				restoreObj.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey])
		}
	}
	return getISCSIPersistentVolumeSource(restoreObj.Spec.VolumeName, namespace, openebsClient)
}

// updateRestoreStatus will update the restore status to given status
func updateRestoreStatus(
	clientset clientset.Interface,
	rst openebsapis.CStorRestore,
	status openebsapis.CStorRestoreStatus) {
	rst.Status = status

	_, err := clientset.OpenebsV1alpha1().CStorRestores(rst.Namespace).Update(&rst)
	if err != nil {
		klog.Errorf("Failed to update restore:%s with status:%v", rst.Name, status)
		return
	}
}

func waitForCVCBoundState(cvcName, namespace string, clientset clientset.Interface) error {
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		cvcObj, err := clientset.CstorV1().CStorVolumeConfigs(namespace).Get(cvcName, metav1.GetOptions{})
		// If CVC is not found then wait for it to exist in etcd
		if err != nil && !k8serror.IsNotFound(err) {
			return err
		}
		if cvcObj.Status.Phase == cstor.CStorVolumeConfigPhaseBound {
			// Which means all the CStorVolume related resources are created
			return nil
		}
		klog.Errorf("waiting for CVC: %s in namespace: %s to become Bounded error: %v", cvcObj.Name, cvcObj.Namespace, err)
		time.Sleep(2 * time.Second)
	}
	return errors.Errorf("CVC %s in namespace %s is not in Bound status", cvcName, namespace)
}

// restoreCreateValidationRequest will validate the restore creation request and
// returns error if any validation failed
func restoreCreateValidationRequest(restore *openebsapis.CStorRestore) error {
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

// getNodeID returns the node name either from node topology only topology is
// specified in the storageclass or from pool manager nodes
func (rOps *restoreAPIOps) getNodeID(scObj *storagev1.StorageClass) (string, error) {
	var labelKey, labelValue string
	cspcName := scObj.Parameters["cstorPoolCluster"]
	if len(scObj.AllowedTopologies) != 0 {
		labelKey = scObj.AllowedTopologies[0].MatchLabelExpressions[0].Key
		labelValue = scObj.AllowedTopologies[0].MatchLabelExpressions[0].Values[0]
	} else {
		cspiList, err := rOps.clientset.CstorV1().
			CStorPoolInstances(rOps.namespace).
			List(metav1.ListOptions{
				LabelSelector: cstortypes.CStorPoolClusterLabelKey + "=" + cspcName,
			})
		if err != nil {
			return "", errors.Wrapf(err, "failed to list CSPI of CSPC %s", cspcName)
		}
		if len(cspiList.Items) == 0 {
			return "", errors.Errorf("no cspi exists for CSPC %s", cspcName)
		}
		labelKey = cstortypes.HostNameLabelKey
		labelValue = cspiList.Items[0].Spec.HostName
	}

	nodeList, err := rOps.k8sclientset.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: labelKey + "=" + labelValue})
	if err != nil {
		return "", errors.Wrapf(err, "falied to list nodes which has %s:%s label", labelKey, labelValue)
	}
	if len(nodeList.Items) == 0 {
		return "", errors.Errorf("no nodes exists for provided label %s:%s", labelKey, labelValue)
	}
	return nodeList.Items[0].Name, nil
}

// buildCStorVolumeConfig build the CVC from StorageClass and restoreObject
// NOTE: This function will be called only in case of local restore request
func (rOps *restoreAPIOps) buildCStorVolumeConfig(
	scObj *storagev1.StorageClass, restoreObj *openebsapis.CStorRestore) (*cstor.CStorVolumeConfig, error) {
	nodeID, err := rOps.getNodeID(scObj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get nodeID")
	}
	replicaCount, _ := strconv.Atoi(scObj.Parameters["replicaCount"])

	// Build CStorVolumeConfig
	// TODO: Convert all literals into constants and make as builder code
	cvcObj := &cstor.CStorVolumeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreObj.Spec.VolumeName,
			Namespace: rOps.namespace,
			Labels: map[string]string{
				cstortypes.CStorPoolClusterLabelKey: scObj.Parameters["cstorPoolCluster"],
				"openebs.io/source-volume":          restoreObj.Spec.VolumeName,
			},
			Annotations: map[string]string{
				"openebs.io/volumeID":      restoreObj.Spec.VolumeName,
				"openebs.io/volume-policy": scObj.Parameters["cstorVolumePolicy"],
			},
			Finalizers: []string{"cvc.openebs.io/finalizer"},
		},
		Spec: cstor.CStorVolumeConfigSpec{
			Provision: cstor.VolumeProvision{
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
		Publish: cstor.CStorVolumeConfigPublish{
			NodeID: nodeID,
		},
		VersionDetails: cstor.VersionDetails{
			Status: cstor.VersionStatus{
				Current: version.Current(),
			},
			Desired: version.Current(),
		},
		Status: cstor.CStorVolumeConfigStatus{
			Phase: cstor.CStorVolumeConfigPhasePending,
		},
	}

	return cvcObj, nil
}

// getISCSIPersistentVolumeSource will return iscsipersistentvolumesource object
// by populating with the help of CStorVolume object
func getISCSIPersistentVolumeSource(
	volumeName, namespace string, clientset clientset.Interface) (interface{}, error) {
	cvObj, err := clientset.CstorV1().CStorVolumes(namespace).Get(volumeName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get %s volume details", volumeName)
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

// getCStorRestoreStatus returns the status of Restore. It returns
// restore status as "Failed" if pool manager (or) pool manager pod
// node is down else it will return same status whatever restore.Status
// contains
func getCStorRestoreStatus(k8sClient kubernetes.Interface,
	rst openebsapis.CStorRestore) openebsapis.CStorRestoreStatus {
	namespace := getOpenEBSNamespace()

	if rst.Status != openebsapis.RSTCStorStatusDone && rst.Status != openebsapis.RSTCStorStatusFailed {
		// check if node is running or not
		bkpNodeDown := checkIfPoolManagerNodeDown(k8sClient, rst.Labels[cstortypes.CStorPoolInstanceNameLabelKey], namespace)
		// check if cstor-pool-mgmt container is running or not
		bkpPodDown := checkIfPoolManagerDown(k8sClient, rst.Labels[cstortypes.CStorPoolInstanceNameLabelKey], namespace)

		if bkpNodeDown || bkpPodDown {
			// Backup is stalled, assume status as failed
			return openebsapis.RSTCStorStatusFailed
		}
	}
	return rst.Status
}
