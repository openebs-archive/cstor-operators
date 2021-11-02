// Copyright 2020 The OpenEBS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webhook

import (
	"context"
	"fmt"
	"net/http"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func (wh *webhook) validatePVC(ar *v1.AdmissionReview) *v1.AdmissionResponse {
	req := ar.Request
	response := &v1.AdmissionResponse{}
	response.Allowed = true
	// validates only if requested operation is CREATE or DELETE
	if req.Operation == v1.Create {
		return wh.validatePVCCreateRequest(req)
	} else if req.Operation == v1.Delete {
		return wh.validatePVCDeleteRequest(req)
	}
	klog.V(2).Infof("Admission wehbook for PVC module not "+
		"configured for %s operation", req.Operation)
	return response
}

// validatePVCDeleteRequest validates the persistentvolumeclaim(PVC) delete request
func (wh *webhook) validatePVCDeleteRequest(req *v1.AdmissionRequest) *v1.AdmissionResponse {
	response := &v1.AdmissionResponse{}
	response.Allowed = true

	// ignore the Delete request of PVC if resource name is empty which
	// can happen as part of cleanup process of namespace
	if req.Name == "" {
		return response
	}

	klog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)

	// TODO* use raw object once validation webhooks support DELETE request
	// object as non nil value https://github.com/kubernetes/kubernetes/issues/66536
	//var pvc corev1.PersistentVolumeClaim
	//err := json.Unmarshal(req.Object.Raw, &pvc)
	//if err != nil {
	//	klog.Errorf("Could not unmarshal raw object: %v, %v", err, req.Object.Raw)
	//	status.Allowed = false
	//	status.Result = &metav1.Status{
	//		Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
	//		Message: err.Error(),
	//	}
	//	return status
	//}

	// fetch the pvc specifications
	pvc, err := wh.kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Get(context.TODO(), req.Name, metav1.GetOptions{})
	if err != nil {
		response.Allowed = false
		response.Result = &metav1.Status{
			Message: fmt.Sprintf("error retrieving PVC: %v", err.Error()),
		}
		return response
	}

	if !validationRequired(ignoredNamespaces, &pvc.ObjectMeta) {
		klog.V(4).Infof("Skipping validation for %s/%s due to policy check", pvc.Namespace, pvc.Name)
		return response
	}

	// construct source-volume label to list all the matched cstorVolumes
	label := fmt.Sprintf("openebs.io/source-volume=%s", pvc.Spec.VolumeName)
	listOptions := metav1.ListOptions{
		LabelSelector: label,
	}

	// get the all CStorVolumes resources in all namespaces based on the
	// source-volume label to verify if there is any clone volume exists.
	// if source-volume label matches with name of PV, failed the pvc
	// deletion operation.

	cStorVolumes, err := wh.getCstorVolumes(listOptions)
	if err != nil {
		response.Allowed = false
		response.Result = &metav1.Status{
			Message: fmt.Sprintf("error retrieving CstorVolumes: %v", err.Error()),
		}
		return response
	}

	if len(cStorVolumes.Items) != 0 {
		response.Allowed = false
		response.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: "PVC with cloned volumes can't be deleted",
			Message: fmt.Sprintf("pvc %q has '%v' cloned volume(s)", pvc.Name, len(cStorVolumes.Items)),
		}
		return response
	}

	cStorVolumeClaims, err := wh.getCstorVolumeClaims(listOptions)
	if err != nil {
		response.Allowed = false
		response.Result = &metav1.Status{
			Message: fmt.Sprintf("error retrieving CstorVolumeClaims: %v", err.Error()),
		}
		return response
	}

	if len(cStorVolumeClaims.Items) != 0 {
		response.Allowed = false
		response.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: "PVC with cloned volumes can't be deleted",
			Message: fmt.Sprintf("pvc %q has '%v' cloned volume(s)", pvc.Name, len(cStorVolumeClaims.Items)),
		}
		return response
	}
	return response
}

// getCstorVolumes gets the list of CstorVolumes based in the source-volume labels
func (wh *webhook) getCstorVolumes(listOptions metav1.ListOptions) (*cstor.CStorVolumeList, error) {
	return wh.clientset.CstorV1().CStorVolumes("").List(context.TODO(), listOptions)
}

// getCstorVolumeClaims gets the list of CstorVolumeclaims based in the source-volume labels
func (wh *webhook) getCstorVolumeClaims(listOptions metav1.ListOptions) (*cstor.CStorVolumeConfigList, error) {
	return wh.clientset.CstorV1().CStorVolumeConfigs("").List(context.TODO(), listOptions)
}

// validatePVCCreateRequest validates persistentvolumeclaim(PVC) create request
func (wh *webhook) validatePVCCreateRequest(req *v1.AdmissionRequest) *v1.AdmissionResponse {
	klog.Infof("Recieved PVC Create Request")
	response := &v1.AdmissionResponse{}
	response.Allowed = true
	// 	var pvc corev1.PersistentVolumeClaim
	// 	err := json.Unmarshal(req.Object.Raw, &pvc)
	// 	if err != nil {
	// 		klog.Errorf("Could not unmarshal raw object: %v, %v", err, req.Object.Raw)
	// 		response.Allowed = false
	// 		response.Result = &metav1.Status{
	// 			Status:  metav1.StatusFailure,
	// 			Code:    http.StatusBadRequest,
	// 			Reason:  metav1.StatusReasonBadRequest,
	// 			Message: err.Error(),
	// 		}
	// 		return response
	// 	}

	// 	// If snapshot.alpha.kubernetes.io/snapshot annotation represents the clone pvc
	// 	// create request
	// 	snapname := pvc.Annotations[snapshotAnnotation]
	// 	if len(snapname) == 0 {
	// 		return response
	// 	}

	// 	klog.V(4).Infof("AdmissionReview for creating a clone volume Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v",
	// 		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)
	// 	// get the snapshot object to get snapshotdata object
	// 	// Note: If snapname is empty then below call will retrun error
	// 	snapObj, err := wh.snapClientSet.OpenebsV1alpha1().VolumeSnapshots(pvc.Namespace).Get(snapname, metav1.GetOptions{})
	// 	if err != nil {
	// 		klog.Errorf("failed to get the snapshot object for snapshot name: '%s' namespace: '%s' PVC: '%s'"+
	// 			"error: '%v'", snapname, pvc.Namespace, pvc.Name, err)
	// 		response.Allowed = false
	// 		response.Result = &metav1.Status{
	// 			Message: fmt.Sprintf("Failed to get the snapshot object for snapshot name: '%s' namespace: '%s' "+
	// 				"error: '%v'", snapname, pvc.Namespace, err.Error()),
	// 		}
	// 		return response
	// 	}

	// 	snapDataName := snapObj.Spec.SnapshotDataName
	// 	if len(snapDataName) == 0 {
	// 		klog.Errorf("Snapshotdata name is empty for snapshot: '%s' snapshot Namespace: '%s' PVC: '%s'",
	// 			snapname, snapObj.ObjectMeta.Namespace, pvc.Name)
	// 		response.Allowed = false
	// 		response.Result = &metav1.Status{
	// 			Message: fmt.Sprintf("Snapshotdata name is empty for snapshot: '%s' snapshot Namespace: '%s'",
	// 				snapname, snapObj.ObjectMeta.Namespace),
	// 		}
	// 		return response
	// 	}
	// 	klog.V(4).Infof("snapshotdata name: '%s'", snapDataName)

	// 	// get the snapDataObj to get the snapshotdataname
	// 	// Note: If snapDataName is empty then below call will return error
	// 	snapDataObj, err := wh.snapClientSet.OpenebsV1alpha1().VolumeSnapshotDatas().Get(snapDataName, metav1.GetOptions{})
	// 	if err != nil {
	// 		klog.Errorf("Failed to get the snapshotdata object for snapshotdata  name: '%s' "+
	// 			"snapName: '%s' namespace: '%s' PVC: '%s' error: '%v'", snapDataName, snapname, snapObj.ObjectMeta.Namespace, pvc.Name, err)
	// 		response.Allowed = false
	// 		response.Result = &metav1.Status{
	// 			Message: fmt.Sprintf("Failed to get the snapshotdata object for snapshotdata  name: '%s' "+
	// 				"snapName: '%s' namespace: '%s' // snapClientSet is a snaphot custom resource package generated from custom API group.
	// 				snapClientSet snapclient.Interfaceerror: '%v'", snapDataName, snapname, snapObj.ObjectMeta.Namespace, err.Error()),
	// 		}
	// 		return response
	// 	}

	// 	snapSizeString := snapDataObj.Spec.OpenEBSSnapshot.Capacity
	// 	// If snapshotdata object doesn't consist Capacity field then we will log it and return false.
	// 	if len(snapSizeString) == 0 {
	// 		klog.Infof("snapshot size not found for snapshot name: '%s' snapshot namespace: '%s' snapshotdata name: '%s'",
	// 			snapname, snapObj.ObjectMeta.Namespace, snapDataName)
	// 		response.Allowed = false
	// 		response.Result = &metav1.Status{
	// 			Message: fmt.Sprintf("PVC: '%s' creation requires upgrade of volumesnapshotdata name: '%s'", pvc.ObjectMeta.Name, snapDataName),
	// 		}
	// 		return response
	// 	}

	// 	snapCapacity := resource.MustParse(snapSizeString)
	// 	pvcSize := pvc.Spec.Resources.Requests[corev1.ResourceName(corev1.ResourceStorage)]
	// 	if pvcSize.Cmp(snapCapacity) != 0 {
	// 		klog.Errorf("Requested pvc size not matched the snapshot size '%s' belongs to snapshot name: '%s' "+
	// 			"snapshot Namespace: '%s' VolumeSnapshotData '%s'", snapSizeString, snapObj.ObjectMeta.Name, snapObj.ObjectMeta.Namespace, snapDataName)
	// 		response.Allowed = false
	// 		response.Result = &metav1.Status{
	// 			Message: fmt.Sprintf("Requested pvc size must be equal to snapshot size '%s' "+
	// 				"which belongs to snapshot name: '%s' snapshot NameSpace: '%s' volumesnapshotdata: '%s'",
	// 				snapSizeString, snapObj.ObjectMeta.Name, snapObj.ObjectMeta.Namespace, snapDataName),
	// 		}
	// 		return response
	// 	}
	return response
}
