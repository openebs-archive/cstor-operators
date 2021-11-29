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
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// v1Alpha1BackupWrapper holds the information
// about v1alpha1 backup resource
type v1Alpha1BackupWrapper struct {
	backup    *openebsapis.CStorBackup
	clientset clientset.Interface
}

func newV1Alpha1BackupWrapper(clientset clientset.Interface) *v1Alpha1BackupWrapper {
	return &v1Alpha1BackupWrapper{
		clientset: clientset}
}

// setBackup sets the v1alpha1 backup in backupWrapper
func (backupWrapper *v1Alpha1BackupWrapper) setBackup(
	backup *openebsapis.CStorBackup) *v1Alpha1BackupWrapper {
	backupWrapper.backup = backup
	return backupWrapper
}

// isBackupCompleted returns true if backup execution is completed
func (backupWrapper *v1Alpha1BackupWrapper) isBackupCompleted() bool {
	if isBackupFailed(backupWrapper.backup) ||
		isBackupSucceeded(backupWrapper.backup) {
		return true
	}
	return false
}

func (backupWrapper *v1Alpha1BackupWrapper) getCSPIName() string {
	return backupWrapper.backup.GetLabels()[cstortypes.CStorPoolInstanceNameLabelKey]
}

func (backupWrapper *v1Alpha1BackupWrapper) findLastBackupStat() string {
	lastbkpname := backupWrapper.backup.Spec.BackupName + "-" + backupWrapper.backup.Spec.VolumeName
	lastbkp, err := backupWrapper.clientset.OpenebsV1alpha1().
		CStorCompletedBackups(backupWrapper.backup.Namespace).
		Get(context.TODO(), lastbkpname, metav1.GetOptions{})
	if err != nil {
		// Unable to fetch the last backup, so we will return fail state
		klog.Errorf("Failed to fetch last completed-backup:%s error:%s", lastbkpname, err.Error())
		return string(openebsapis.BKPCStorStatusFailed)
	}

	// lastbkp stores the last(PrevSnapName) and 2nd last(SnapName) completed snapshot
	// let's check if last backup's snapname/PrevSnapName  matches with current snapshot name
	if backupWrapper.backup.Spec.SnapName == lastbkp.Spec.SnapName ||
		backupWrapper.backup.Spec.SnapName == lastbkp.Spec.PrevSnapName {
		return string(openebsapis.BKPCStorStatusDone)
	}

	// lastbackup snap/prevsnap doesn't match with bkp snapname
	return string(openebsapis.BKPCStorStatusFailed)
}

func (backupWrapper *v1Alpha1BackupWrapper) updateBackupStatus(
	backupStatus string) backupHelper {
	backupWrapper.backup.Status = openebsapis.CStorBackupStatus(backupStatus)

	_, err := backupWrapper.clientset.
		OpenebsV1alpha1().
		CStorBackups(backupWrapper.backup.Namespace).Update(context.TODO(), backupWrapper.backup, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Failed to update backup:%s with status:%v", backupWrapper.backup.Name, backupStatus)
	}
	return backupWrapper
}

func (backupWrapper *v1Alpha1BackupWrapper) deleteCompletedBackup(name, namespace, snapName string) error {
	// Let's get the cstorCompletedBackup resource for the given backup
	// CStorCompletedBackups resource stores the information about last two completed snapshots
	lastbkp, err := backupWrapper.clientset.
		OpenebsV1alpha1().
		CStorCompletedBackups(namespace).
		Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serror.IsNotFound(err) {
		return errors.Wrapf(err, "failed to fetch last-completed-backup=%s resource", name)
	}

	// lastbkp stores the last(PrevSnapName) and 2nd last(SnapName) completed snapshot
	// If given backup is the last backup of scheduled backup (lastbkp.Spec.PrevSnapName == backup) or
	// completedBackup doesn't have successful backup(len(lastbkp.Spec.PrevSnapName) == 0) then we will delete the lastbkp CR
	// Deleting this CR make sure that next backup of the schedule will be full backup
	if lastbkp != nil && (lastbkp.Spec.PrevSnapName == snapName || len(lastbkp.Spec.PrevSnapName) == 0) {
		err := backupWrapper.clientset.OpenebsV1alpha1().CStorCompletedBackups(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil && !k8serror.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete last-completed-backup=%s resource", name)
		}
	}
	return nil
}

func (backupWrapper *v1Alpha1BackupWrapper) deleteBackup(name, namespace string) error {
	err := backupWrapper.clientset.OpenebsV1alpha1().CStorBackups(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !k8serror.IsNotFound(err) {
		return errors.Wrapf(err, "failed to delete cstorbackup: %s resource", name)
	}
	return nil
}

func (backupWrapper *v1Alpha1BackupWrapper) getBackupObject() interface{} {
	return backupWrapper.backup
}

func (backupWrapper *v1Alpha1BackupWrapper) getOrCreateLastBackupSnap() (string, error) {
	lastbkpName := backupWrapper.backup.Spec.BackupName + "-" + backupWrapper.backup.Spec.VolumeName

	// When only few pools of CStorPoolCluster is upgrade and if the backup request is scheduled
	// backup then we need to check for v1 version of completed backup to get last snapshot name
	completedBackup, err := backupWrapper.clientset.CstorV1().
		CStorCompletedBackups(backupWrapper.backup.Namespace).
		Get(context.TODO(), lastbkpName, metav1.GetOptions{})
	if err != nil && !k8serror.IsNotFound(err) {
		return "", errors.Wrapf(err, "failed to get v1 completed backup %s", lastbkpName)
	}
	if err == nil {
		return completedBackup.Spec.LastSnapName, nil
	}

	b, err := backupWrapper.clientset.OpenebsV1alpha1().
		CStorCompletedBackups(backupWrapper.backup.Namespace).
		Get(context.TODO(), lastbkpName, metav1.GetOptions{})
	if err != nil {
		if k8serror.IsNotFound(err) {
			// Build CStorCompletedBackup which will helpful for incremental backups
			bk := &openebsapis.CStorCompletedBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      lastbkpName,
					Namespace: backupWrapper.backup.Namespace,
					Labels:    backupWrapper.backup.Labels,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: backupWrapper.backup.Spec.BackupName,
					VolumeName: backupWrapper.backup.Spec.VolumeName,
				},
			}

			_, err := backupWrapper.clientset.OpenebsV1alpha1().CStorCompletedBackups(bk.Namespace).Create(context.TODO(), bk, metav1.CreateOptions{})
			if err != nil {
				klog.Errorf("Error creating last completed-backup resource for backup:%v err:%v", bk.Spec.BackupName, err)
				return "", err
			}
			klog.Infof("LastBackup resource created for backup:%s volume:%s", bk.Spec.BackupName, bk.Spec.VolumeName)
			return "", nil
		}
		return "", errors.Errorf("failed to get lastbkpName %s error: %s", lastbkpName, err.Error())
	}

	// PrevSnapName stores the last completed backup snapshot
	return b.Spec.PrevSnapName, nil
}

func (backupWrapper *v1Alpha1BackupWrapper) setBackupStatus(status string) backupHelper {
	backupWrapper.backup.Status = openebsapis.CStorBackupStatus(status)
	return backupWrapper
}

func (backupWrapper *v1Alpha1BackupWrapper) setLastSnapshotName(snapName string) backupHelper {
	backupWrapper.backup.Spec.PrevSnapName = snapName
	return backupWrapper
}

func (backupWrapper *v1Alpha1BackupWrapper) createBackupResource() (backupHelper, error) {
	_, err := backupWrapper.clientset.OpenebsV1alpha1().
		CStorBackups(backupWrapper.backup.Namespace).
		Create(context.TODO(), backupWrapper.backup, metav1.CreateOptions{})
	if err != nil {
		klog.Errorf("Failed to create backup: error '%s'", err.Error())
		return backupWrapper, errors.Wrapf(err, "failed to create backup %s", backupWrapper.backup.Name)
	}
	return backupWrapper, nil
}

// isBackupFailed returns true if backup failed
func isBackupFailed(backup *openebsapis.CStorBackup) bool {
	return backup.Status == openebsapis.BKPCStorStatusFailed
}

// isBackupSucceeded returns true if backup completed successfully
func isBackupSucceeded(backup *openebsapis.CStorBackup) bool {
	return backup.Status == openebsapis.BKPCStorStatusDone
}
