/*
Copyright 2018 The OpenEBS Authors.

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

package backupcontroller

import (
	"context"
	"fmt"

	cstorapis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/cstor-operators/pkg/controllers/common"
	"github.com/openebs/cstor-operators/pkg/volumereplica"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the CStorBackup resource
// with the current status of the resource.
func (c *BackupController) syncHandler(key string, operation common.QueueOperation) error {
	bkp, err := c.getCStorBackupResource(key)
	if err != nil {
		return err
	}
	if bkp == nil {
		return fmt.Errorf("cannot retrieve CStorBackup %q", key)
	}
	if bkp.IsSucceeded() || bkp.IsFailed() {
		return nil
	}

	status, err := c.eventHandler(operation, bkp)
	if err != nil {
		klog.Errorf(err.Error())
		bkp.Status = cstorapis.BKPCStorStatusFailed
	} else {
		bkp.Status = cstorapis.CStorBackupStatus(status)
	}
	if status == "" {
		return nil
	}

	nbkp, err := c.clientset.CstorV1().CStorBackups(bkp.Namespace).Get(context.TODO(), bkp.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	nbkp.Status = bkp.Status

	_, err = c.clientset.CstorV1().CStorBackups(nbkp.Namespace).Update(context.TODO(), nbkp, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	klog.Infof("Completed operation:%v for backup:%v, status:%v", operation, nbkp.Name, nbkp.Status)
	return nil
}

// eventHandler will execute a function according to a given operation
func (c *BackupController) eventHandler(operation common.QueueOperation, bkp *cstorapis.CStorBackup) (string, error) {
	klog.Infof("%s operation on Backup %s", operation, bkp.Name)
	switch operation {
	case common.QOpAdd:
		return c.addEventHandler(bkp)
	case common.QOpDestroy:
		/* TODO: Handle backup destroy event
		 */
		return "", nil
	case common.QOpSync:
		return c.syncEventHandler(bkp)
	}
	return string(cstorapis.BKPCStorStatusInvalid), nil
}

// addEventHandler will change the state of backup to Init state.
func (c *BackupController) addEventHandler(bkp *cstorapis.CStorBackup) (string, error) {
	if !bkp.IsPending() {
		return string(cstorapis.BKPCStorStatusInvalid), nil
	}
	c.recorder.Event(bkp, corev1.EventTypeNormal, "Update", "initilized backup process")
	return string(cstorapis.BKPCStorStatusInit), nil
}

// syncEventHandler will perform the backup if a given backup is in init state
func (c *BackupController) syncEventHandler(bkp *cstorapis.CStorBackup) (string, error) {
	// If the backup is in init state then only we will complete the backup
	if bkp.IsInInit() {
		bkp.Status = cstorapis.BKPCStorStatusInProgress
		_, err := c.clientset.CstorV1().CStorBackups(bkp.Namespace).Update(context.TODO(), bkp, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Failed to update backup:%s status : %v", bkp.Name, err.Error())
			return "", err
		}

		err = volumereplica.CreateVolumeBackup(bkp)
		if err != nil {
			c.recorder.Eventf(bkp, corev1.EventTypeWarning, "Backup", "failed to create backup error: %s", err.Error())
			return string(cstorapis.BKPCStorStatusFailed), err
		}

		c.recorder.Event(bkp, corev1.EventTypeNormal, "Backup", "backup creation is successful")
		klog.Infof("backup creation successful: %v, %v", bkp.ObjectMeta.Name, string(bkp.GetUID()))
		err = c.updateCStorCompletedBackup(bkp)
		if err != nil {
			return string(cstorapis.BKPCStorStatusFailed), err
		}
		return string(cstorapis.BKPCStorStatusDone), nil
	}
	return "", nil
}

// getCStorBackupResource returns a backup object corresponding to the resource key
func (c *BackupController) getCStorBackupResource(key string) (*cstorapis.CStorBackup, error) {
	// Convert the key(namespace/name) string into a distinct name
	klog.V(1).Infof("Finding backup for key:%s", key)
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil, nil
	}

	bkp, err := c.clientset.CstorV1().CStorBackups(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("bkp '%s' in work queue no longer exists", key))
			return nil, nil
		}
		return nil, err
	}
	return bkp, nil
}

// IsDestroyEvent is to check if the call is for backup destroy.
func IsDestroyEvent(bkp *cstorapis.CStorBackup) bool {
	return bkp.ObjectMeta.DeletionTimestamp != nil
}

// updateCStorCompletedBackup updates the CStorCompletedBackups resource for the given backup
// CStorCompletedBackups stores the information of last two completed backups
// For example, if schedule `b` has last two backups b-0 and b-1 (b-0 created first and after that b-1 was created) having snapshots
// b-0 and b-1 respectively then CStorCompletedBackups for the schedule `b` will have following information :
//	CStorCompletedBackups.Spec.PrevSnapName =  b-1
//  CStorCompletedBackups.Spec.SnapName = b-0
func (c *BackupController) updateCStorCompletedBackup(bkp *cstorapis.CStorBackup) error {
	lastbkpname := bkp.Spec.BackupName + "-" + bkp.Spec.VolumeName

	// There can be cases where only few pools of CSPC is upgraded but not all
	// cstor pools. In such cases if backup is for scheduled backup then and completed
	// backup belongs to the pool that supports v1alpha1 then we have to update v1alpha1
	// completed backup resource
	completedBackup, err := c.clientset.OpenebsV1alpha1().CStorCompletedBackups(bkp.Namespace).Get(context.TODO(), lastbkpname, metav1.GetOptions{})
	if err != nil && !k8serror.IsNotFound(err) {
		klog.Errorf("failed to get completed backup for %s vol: %v error: %v", bkp.Spec.BackupName, bkp.Spec.VolumeName, err)
	}
	// Update v1alpha1 backup resource if exist
	if err == nil {
		// SnapName store the name of 2nd last backed up snapshot
		completedBackup.Spec.SnapName = completedBackup.Spec.PrevSnapName

		// PrevSnapName store the name of last backed up snapshot<Paste>
		completedBackup.Spec.PrevSnapName = bkp.Spec.SnapName

		_, err = c.clientset.OpenebsV1alpha1().CStorCompletedBackups(bkp.Namespace).Update(context.TODO(), completedBackup, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Failed to update lastbackup for %s", completedBackup.Name)
			return err
		}
		return nil
	}

	bkplast, err := c.clientset.CstorV1().CStorCompletedBackups(bkp.Namespace).Get(context.TODO(), lastbkpname, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get last completed backup for %s vol:%v error: %v", bkp.Spec.BackupName, bkp.Spec.VolumeName, err)
		return nil
	}

	// SecondLastSnapName store the name of 2nd last backed up snapshot
	bkplast.Spec.SecondLastSnapName = bkplast.Spec.LastSnapName

	// LastSnapName store the name of last backed up snapshot
	bkplast.Spec.LastSnapName = bkp.Spec.SnapName
	_, err = c.clientset.CstorV1().CStorCompletedBackups(bkp.Namespace).Update(context.TODO(), bkplast, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Failed to update lastbackup for %s", bkplast.Name)
		return err
	}

	return nil
}
