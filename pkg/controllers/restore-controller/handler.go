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

package restorecontroller

import (
	"context"
	"fmt"
	"os"
	"reflect"

	cstorapis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/cstor-operators/pkg/controllers/common"
	"github.com/openebs/cstor-operators/pkg/volumereplica"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the CStorReplicaUpdated resource
// with the current status of the resource.
func (c *RestoreController) syncHandler(key string, operation common.QueueOperation) error {
	klog.Infof("Sync handler called for key:%s with op:%s", key, operation)
	rst, err := c.getCStorRestoreResource(key)
	if err != nil {
		return err
	}
	if rst == nil {
		return fmt.Errorf("can not retrieve CStorRestore %q", key)
	}

	if rst.IsSucceeded() || rst.IsFailed() {
		return nil
	}

	status, err := c.rstEventHandler(operation, rst)
	if status == "" {
		return nil
	}

	if err != nil {
		klog.Errorf(err.Error())
		rst.Status = cstorapis.RSTCStorStatusFailed
	} else {
		rst.Status = cstorapis.CStorRestoreStatus(status)
	}

	nrst, err := c.clientset.CstorV1().CStorRestores(rst.Namespace).Get(context.TODO(), rst.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	nrst.Status = rst.Status

	_, err = c.clientset.CstorV1().CStorRestores(nrst.Namespace).Update(context.TODO(), nrst, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	klog.Infof("Completed operation:%v for restore:%v, status:%v", operation, nrst.Name, nrst.Status)
	return nil
}

// eventHandler will execute a function according to a given operation
func (c *RestoreController) rstEventHandler(operation common.QueueOperation, rst *cstorapis.CStorRestore) (string, error) {
	switch operation {
	case common.QOpAdd:
		return c.addEventHandler(rst)
	case common.QOpDestroy:
		/*
			status, err := c.rstDestroyEventHandler(rstGot)
			return status, err
			klog.Infof("Processing restore delete event %v, %v", rstGot.ObjectMeta.Name, string(rstGot.GetUID()))
		*/
		return "", nil
	case common.QOpSync:
		return c.syncEventHandler(rst)
	case common.QOpModify:
		return "", nil
	}
	return string(cstorapis.RSTCStorStatusInvalid), nil
}

// addEventHandler will change the state of restore to Init state.
func (c *RestoreController) addEventHandler(rst *cstorapis.CStorRestore) (string, error) {
	if !rst.IsPending() {
		return string(cstorapis.RSTCStorStatusInvalid), nil
	}
	return string(cstorapis.RSTCStorStatusInit), nil
}

// syncEventHandler will perform the restore if a given restore is in init state
func (c *RestoreController) syncEventHandler(rst *cstorapis.CStorRestore) (string, error) {
	// If the restore is in init state then only we will complete the restore
	if rst.IsInInit() {
		rst.Status = cstorapis.RSTCStorStatusInProgress
		_, err := c.clientset.CstorV1().CStorRestores(rst.Namespace).Update(context.TODO(), rst, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Failed to update restore:%s status : %v", rst.Name, err.Error())
			return "", err
		}

		err = volumereplica.CreateVolumeRestore(rst)
		if err != nil {
			klog.Errorf("restore creation failure: %v", err.Error())
			return string(cstorapis.RSTCStorStatusFailed), err
		}
		c.recorder.Event(rst, corev1.EventTypeNormal, string(common.SuccessCreated), string(common.MessageResourceCreated))
		klog.Infof("restore creation successful: %v, %v", rst.ObjectMeta.Name, string(rst.GetUID()))
		return string(cstorapis.RSTCStorStatusDone), nil
	} else if rst.IsPending() {
		klog.Infof("Updating restore:%s status to %v", rst.Name, cstorapis.RSTCStorStatusInit)
		return string(cstorapis.RSTCStorStatusInit), nil
	}
	return "", nil
}

// getCStorRestoreResource returns a restore object corresponding to the resource key
func (c *RestoreController) getCStorRestoreResource(key string) (*cstorapis.CStorRestore, error) {
	// Convert the key(namespace/name) string into a distinct name
	klog.V(1).Infof("Finding restore for key:%s", key)
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil, nil
	}

	rst, err := c.clientset.CstorV1().CStorRestores(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Restore resource for key:%s is missing", name)
		return nil, err
	}
	return rst, nil
}

// IsRightCStorPoolMgmt is to check if the restore request is for particular pod/application.
func IsRightCStorPoolMgmt(rst *cstorapis.CStorRestore) bool {
	return os.Getenv(string(common.OpenEBSIOCSPIID)) == rst.ObjectMeta.Labels[types.CStorPoolInstanceUIDLabelKey]
}

// IsDestroyEvent is to check if the call is for restore destroy.
func IsDestroyEvent(rst *cstorapis.CStorRestore) bool {
	return rst.ObjectMeta.DeletionTimestamp != nil
}

// IsOnlyStatusChange is to check only status change of restore object.
func IsOnlyStatusChange(oldrst, newrst *cstorapis.CStorRestore) bool {
	if reflect.DeepEqual(oldrst.Spec, newrst.Spec) &&
		!reflect.DeepEqual(oldrst.Status, newrst.Status) {
		return true
	}
	return false
}
