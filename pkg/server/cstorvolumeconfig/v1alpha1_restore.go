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

	openebsapis "github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	cstortypes "github.com/openebs/api/v3/pkg/apis/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func (rOps *restoreAPIOps) getV1Alpha1CStorRestoreStatus(
	restoreList *openebsapis.CStorRestoreList) openebsapis.CStorRestoreStatus {
	rstStatus := openebsapis.RSTCStorStatusEmpty
	namespace := getOpenEBSNamespace()

	for _, restore := range restoreList.Items {
		rstStatus = restore.Status
		if restore.Status != openebsapis.RSTCStorStatusDone &&
			restore.Status != openebsapis.RSTCStorStatusFailed {
			poolName := restore.Labels[cstortypes.CStorPoolInstanceNameLabelKey]
			isPoolDown := isPoolManagerDown(rOps.k8sclientset, poolName, namespace)
			if isPoolDown {
				rstStatus = openebsapis.RSTCStorStatusFailed
			}
		}

		switch rstStatus {
		case openebsapis.RSTCStorStatusInProgress:
			rstStatus = openebsapis.RSTCStorStatusInProgress
		case openebsapis.RSTCStorStatusFailed, openebsapis.RSTCStorStatusInvalid:
			if restore.Status != rstStatus {
				// Restore for given CVR may failed due to node failure or pool failure
				// Let's update status for given CVR's restore to failed
				restore.Status = rstStatus
				_, err := rOps.clientset.OpenebsV1alpha1().CStorRestores(restore.Namespace).Update(context.TODO(), &restore, metav1.UpdateOptions{})
				if err != nil {
					klog.Errorf("Failed to update restore:%s with status:%v", restore.Name, rstStatus)
				}
				rstStatus = openebsapis.RSTCStorStatusFailed
			}
		case openebsapis.RSTCStorStatusDone:
			if rstStatus != openebsapis.RSTCStorStatusFailed {
				rstStatus = openebsapis.RSTCStorStatusDone
			}
		}

		klog.Infof("Restore:%v status is %v", restore.Name, restore.Status)

		if rstStatus == openebsapis.RSTCStorStatusInProgress {
			break
		}
	}
	return rstStatus
}
