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
	"net/http"

	"encoding/json"
	"fmt"
	"strings"

	cstorapis "github.com/openebs/api/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/pkg/apis/openebs.io/v1alpha1"
	cstortypes "github.com/openebs/api/pkg/apis/types"
	clientset "github.com/openebs/api/pkg/client/clientset/versioned"
	"github.com/openebs/api/pkg/util"
	snapshot "github.com/openebs/cstor-operators/pkg/snapshot"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// backupAPIOps holds the clients, http request and
// response
type backupAPIOps struct {
	req          *http.Request
	resp         http.ResponseWriter
	k8sclientset kubernetes.Interface
	clientset    clientset.Interface
	snapshoter   snapshot.Snapshoter
}

var (
	// openebsNamespace is the namespace where openebs is deployed
	openebsNamespace string
)

// backupV1alpha1SpecificRequest deals with backup API requests
func (s *HTTPServer) backupV1alpha1SpecificRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	backupOp := &backupAPIOps{
		req:          req,
		resp:         resp,
		k8sclientset: s.cvcServer.kubeclientset,
		clientset:    s.cvcServer.clientset,
		snapshoter:   s.cvcServer.snapshoter,
	}

	switch req.Method {
	case "POST":
		return backupOp.create()
	case "GET":
		klog.Infof("Got backup GET request")
		return backupOp.get()
	case "DELETE":
		klog.Infof("Got backup DELETE request")
		return backupOp.delete()
	}
	return nil, CodedError(405, ErrInvalidMethod)
}

// Create is http handler which handles backup create request
func (bOps *backupAPIOps) create() (interface{}, error) {
	backUp := &openebsapis.CStorBackup{}

	err := decodeBody(bOps.req, backUp)
	if err != nil {
		return nil, err
	}

	if err := backupCreateRequestValidations(backUp); err != nil {
		return nil, err
	}

	// TODO: Move this to interface so that we can mock
	// snapshot calls
	snapshot := snapshot.Snapshot{
		VolumeName:   backUp.Spec.VolumeName,
		SnapshotName: backUp.Spec.SnapName,
		Namespace:    getOpenEBSNamespace(),
		SnapClient:   bOps.snapshoter,
	}
	snapResp, err := snapshot.CreateSnapshot(bOps.clientset)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to create snapshot '%v'", err))
	}
	klog.Infof("Backup snapshot:'%s' created successfully for volume:%s response: %s", backUp.Spec.SnapName, backUp.Spec.VolumeName, snapResp)

	// In case of local backup no need to create CStorBackup CR
	if backUp.Spec.LocalSnap {
		return "", nil
	}

	backUp.Name = backUp.Spec.SnapName + "-" + backUp.Spec.VolumeName

	// find healthy CVR which will helps to create backup CR
	cvr, err := findHealthyCVR(bOps.clientset, backUp.Spec.VolumeName)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to find healthy replica"))
	}

	backUp.ObjectMeta.Labels = map[string]string{
		cstortypes.CStorPoolInstanceUIDLabelKey:  cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceUIDLabelKey],
		cstortypes.CStorPoolInstanceNameLabelKey: cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey],
		cstortypes.PersistentVolumeLabelKey:      cvr.ObjectMeta.Labels[cstortypes.PersistentVolumeLabelKey],
		"openebs.io/backup":                      backUp.Spec.BackupName,
	}

	// Find last backup snapshot name
	lastsnap, err := bOps.getLastBackupSnap(backUp)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed create lastbackup"))
	}

	// Initialize backup status as pending
	backUp.Status = openebsapis.BKPCStorStatusPending
	backUp.Spec.PrevSnapName = lastsnap

	klog.Infof("Creating backup %s for volume %q poolName: %v poolUUID:%v", backUp.Spec.SnapName,
		backUp.Spec.VolumeName,
		backUp.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey],
		backUp.ObjectMeta.Labels[cstortypes.CStorPoolInstanceUIDLabelKey])

	_, err = bOps.clientset.OpenebsV1alpha1().CStorBackups(backUp.Namespace).Create(backUp)
	if err != nil {
		klog.Errorf("Failed to create backup: error '%s'", err.Error())
		return nil, CodedError(500, err.Error())
	}

	klog.Infof("Backup resource:'%s' created successfully", backUp.Name)
	return "", nil
}

// get is http handler which handles backup get request
// It will get the snapshot created by the given backup if backup is done/failed
func (bOps *backupAPIOps) get() (interface{}, error) {
	backUp := &openebsapis.CStorBackup{}

	err := decodeBody(bOps.req, backUp)
	if err != nil {
		return nil, err
	}

	// backup name is expected
	if len(strings.TrimSpace(backUp.Spec.BackupName)) == 0 {
		return nil, CodedError(400, fmt.Sprintf("Failed to get backup: missing backup name "))
	}

	// namespace is expected
	if len(strings.TrimSpace(backUp.Namespace)) == 0 {
		return nil, CodedError(400, fmt.Sprintf("Failed to get backup '%v': missing namespace", backUp.Spec.BackupName))
	}

	// volume name is expected
	if len(strings.TrimSpace(backUp.Spec.VolumeName)) == 0 {
		return nil, CodedError(400, fmt.Sprintf("Failed to get backup '%v': missing volume name", backUp.Spec.BackupName))
	}

	backUp.Name = backUp.Spec.SnapName + "-" + backUp.Spec.VolumeName
	backUpObj, err := bOps.clientset.OpenebsV1alpha1().
		CStorBackups(backUp.Namespace).
		Get(backUp.Name, metav1.GetOptions{})
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to fetch backup error:%v", err))
	}

	if !isBackupCompleted(backUpObj) {
		cspiName := backUpObj.Labels[cstortypes.CStorPoolInstanceNameLabelKey]
		// check if node is running or not
		backUpNodeDown := checkIfPoolManagerNodeDown(bOps.k8sclientset, cspiName)
		// check if cstor-pool-mgmt container is running or not
		backUpPodDown := checkIfPoolManagerDown(bOps.k8sclientset, cspiName)

		if backUpNodeDown || backUpPodDown {
			// Backup is stalled, let's find last completed-backup status
			laststat := findLastBackupStat(bOps.clientset, backUpObj)
			// Update Backup status according to last completed-backup
			updateBackupStatus(bOps.clientset, backUpObj, laststat)

			// Get updated backup object
			backUpObj, err = bOps.clientset.OpenebsV1alpha1().CStorBackups(backUp.Namespace).Get(backUp.Name, metav1.GetOptions{})
			if err != nil {
				return nil, CodedError(400, fmt.Sprintf("Failed to fetch backup error:%v", err))
			}
		}
	}

	out, err := json.Marshal(backUpObj)
	if err == nil {
		_, err = bOps.resp.Write(out)
		if err != nil {
			return nil, CodedError(400, fmt.Sprintf("Failed to send response data"))
		}
		return nil, nil
	}

	return nil, CodedError(400, fmt.Sprintf("Failed to encode response data"))
}

// delete is http handler which handles backup delete request
func (bOps *backupAPIOps) delete() (interface{}, error) {
	// Extract name of backup from path after trimming
	backup := strings.TrimSpace(strings.TrimPrefix(bOps.req.URL.Path, "/latest/backups/"))

	// volname is the volume name in the query params
	volname := bOps.req.URL.Query().Get("volume")

	// namespace is the namespace(pvc namespace) name in the query params
	namespace := bOps.req.URL.Query().Get("namespace")

	// schedule name is the schedule name for the given backup, for non-scheduled backup it will be backup name
	scheduleName := bOps.req.URL.Query().Get("schedule")

	if len(backup) == 0 || len(volname) == 0 || len(namespace) == 0 || len(scheduleName) == 0 {
		return nil, CodedError(405, "failed to delete backup: Insufficient info -- required values volume_name, backup_name, namespace, schedule_name")
	}

	klog.Infof("Deleting backup=%s for volume=%s in namesace=%s and schedule=%s", backup, volname, namespace, scheduleName)

	err := bOps.deleteBackup(backup, volname, namespace, scheduleName)
	if err != nil {
		klog.Errorf("Error deleting backup=%s for volume=%s in namesace=%s and schedule=%s..  error=%s", backup, volname, namespace, scheduleName, err.Error())
		return nil, CodedError(500, fmt.Sprintf("Error deleting backup=%s for volume=%s with namesace=%s and schedule=%s..  error=%s", backup, volname, namespace, scheduleName, err))
	}
	return "", nil
}

// deleteBackup delete the relevant cstorBackup/cstorCompletedBackup resource and cstor snapshot for the given backup
func (bOps *backupAPIOps) deleteBackup(backup, volname, ns, schedule string) error {
	lastCompletedBackup := schedule + "-" + volname

	// Let's get the cstorCompletedBackup resource for the given backup
	// CStorCompletedBackups resource stores the information about last two completed snapshots
	lastbkp, err := bOps.clientset.
		OpenebsV1alpha1().
		CStorCompletedBackups(ns).
		Get(lastCompletedBackup, metav1.GetOptions{})
	if err != nil && !k8serror.IsNotFound(err) {
		return errors.Wrapf(err, "failed to fetch last-completed-backup=%s resource", lastCompletedBackup)
	}

	// lastbkp stores the last(PrevSnapName) and 2nd last(SnapName) completed snapshot
	// If given backup is the last backup of scheduled backup (lastbkp.Spec.PrevSnapName == backup) or
	// completedBackup doesn't have successful backup(len(lastbkp.Spec.PrevSnapName) == 0) then we will delete the lastbkp CR
	// Deleting this CR make sure that next backup of the schedule will be full backup
	if lastbkp != nil && (lastbkp.Spec.PrevSnapName == backup || len(lastbkp.Spec.PrevSnapName) == 0) {
		err := bOps.clientset.OpenebsV1alpha1().CStorCompletedBackups(ns).Delete(lastCompletedBackup, &metav1.DeleteOptions{})
		if err != nil && !k8serror.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete last-completed-backup=%s resource", lastCompletedBackup)
		}
	}

	snapshot := snapshot.Snapshot{
		VolumeName:   volname,
		SnapshotName: backup,
		Namespace:    ns,
		SnapClient:   bOps.snapshoter,
	}
	// Snapshot Name and backup name are same
	_, err = snapshot.DeleteSnapshot(bOps.clientset)
	if err != nil {
		return errors.Wrapf(err, "snapshot deletion failed")
	}

	cstorBackup := backup + "-" + volname
	err = bOps.clientset.OpenebsV1alpha1().CStorBackups(ns).Delete(cstorBackup, &metav1.DeleteOptions{})
	if err != nil && !k8serror.IsNotFound(err) {
		return errors.Wrapf(err, "failed to delete cstorbackup: %s resource", cstorBackup)
	}
	return nil
}

// backupCreateRequestValidations validates the backup create request
func backupCreateRequestValidations(backUp *openebsapis.CStorBackup) error {
	// backup name is expected
	if len(strings.TrimSpace(backUp.Spec.BackupName)) == 0 {
		return CodedError(400, fmt.Sprintf("Failed to create backup: missing backup name "))
	}

	// namespace is expected
	if len(strings.TrimSpace(backUp.Namespace)) == 0 {
		return CodedError(400, fmt.Sprintf("Failed to create backup '%v': missing namespace", backUp.Spec.BackupName))
	}

	// volume name is expected
	if len(strings.TrimSpace(backUp.Spec.VolumeName)) == 0 {
		return CodedError(400, fmt.Sprintf("Failed to create backup '%v': missing volume name", backUp.Spec.BackupName))
	}

	// backupIP is expected for remote snapshot
	if !backUp.Spec.LocalSnap && len(strings.TrimSpace(backUp.Spec.BackupDest)) == 0 {
		return CodedError(400, fmt.Sprintf("Failed to create backup '%v': missing backupIP", backUp.Spec.BackupName))
	}

	// snapshot name is expected
	if len(strings.TrimSpace(backUp.Spec.SnapName)) == 0 {
		return CodedError(400, fmt.Sprintf("Failed to create backup '%v': missing snapName", backUp.Spec.BackupName))
	}
	return nil
}

// getVolumeIP fetches the cstor target service IP Address
func getVolumeIP(volumeName string, clientset clientset.Interface) (string, error) {
	namespace := getOpenEBSNamespace()

	// Fetch the corresponding cstorvolume
	cstorvolume, err := clientset.CstorV1().CStorVolumes(namespace).
		Get(volumeName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return cstorvolume.Spec.TargetIP, nil
}

// getOpenEBSNamespace returns namespace where
// cvc operator is running
func getOpenEBSNamespace() string {
	if openebsNamespace == "" {
		openebsNamespace = util.GetEnv(util.OpenEBSNamespace)
	}
	return openebsNamespace
}

// findHealthyCVR returns Heathy CVR if exists else
// it will return error
func findHealthyCVR(
	openebsClient clientset.Interface,
	volume string) (*cstorapis.CStorVolumeReplica, error) {
	namespace := getOpenEBSNamespace()
	listOptions := metav1.ListOptions{
		LabelSelector: "openebs.io/persistent-volume=" + volume,
	}

	cvrList, err := openebsClient.CstorV1().CStorVolumeReplicas(namespace).List(listOptions)
	if err != nil {
		return nil, err
	}

	// Select a healthy cvr for backup
	for _, cvr := range cvrList.Items {
		if cvr.Status.Phase == cstorapis.CVRStatusOnline {
			return &cvr, nil
		}
	}

	return nil, errors.New("unable to find healthy CVR")
}

// getLastBackupSnap will fetch the last successful backup's snapshot name
func (bOps *backupAPIOps) getLastBackupSnap(backUp *openebsapis.CStorBackup) (string, error) {
	lastbkpName := backUp.Spec.BackupName + "-" + backUp.Spec.VolumeName
	b, err := bOps.clientset.OpenebsV1alpha1().
		CStorCompletedBackups(backUp.Namespace).
		Get(lastbkpName, metav1.GetOptions{})
	if err != nil {
		if k8serror.IsNotFound(err) {
			// Build CStorCompletedBackup which will helpful for incremental backups
			bk := &openebsapis.CStorCompletedBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      lastbkpName,
					Namespace: backUp.Namespace,
					Labels:    backUp.Labels,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: backUp.Spec.BackupName,
					VolumeName: backUp.Spec.VolumeName,
				},
			}

			_, err := bOps.clientset.OpenebsV1alpha1().CStorCompletedBackups(bk.Namespace).Create(bk)
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

// checkIfPoolManagerNodeDown will check if CSPI pool manager is in running or not
func checkIfPoolManagerNodeDown(k8sclient kubernetes.Interface, cspiName string) bool {
	var nodeDown = true
	var pod *corev1.Pod
	var err error

	// If cspiName is not empty then fetch the CStor pool pod using CSPI name
	if cspiName == "" {
		klog.Errorf("failed to find pool manager empty CSPI is provided")
		return nodeDown
	}
	pod, err = fetchPoolManagerFromCSPI(k8sclient, cspiName)
	if err != nil {
		klog.Errorf("Failed to find pool manager for CSPI:%s err:%s", cspiName, err.Error())
		return nodeDown
	}

	if pod.Spec.NodeName == "" {
		klog.Errorf("node name is empty in pool manager %s", pod.Name)
		return nodeDown
	}

	node, err := k8sclient.CoreV1().Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		klog.Infof("Failed to fetch node info for CSPI:%s: %v", cspiName, err)
		return nodeDown
	}
	for _, nodestat := range node.Status.Conditions {
		if nodestat.Type == corev1.NodeReady && nodestat.Status != corev1.ConditionTrue {
			klog.Infof("Node:%v is not in ready state", node.Name)
			return nodeDown
		}
	}
	return !nodeDown
}

// checkIfPoolManagerDown will check if pool pod is running or not
func checkIfPoolManagerDown(k8sclient kubernetes.Interface, cspiName string) bool {
	var podDown = true
	var pod *corev1.Pod
	var err error

	// If cspiName is not empty then fetch the CStor pool pod using CSPI name
	if cspiName == "" {
		klog.Errorf("failed to find pool manager empty CSPI is provided")
		return podDown
	}
	pod, err = fetchPoolManagerFromCSPI(k8sclient, cspiName)
	if err != nil {
		klog.Errorf("Failed to find pool manager for CSPI:%s err:%s", cspiName, err.Error())
		return podDown
	}

	for _, containerstatus := range pod.Status.ContainerStatuses {
		if containerstatus.Name == "cstor-pool-mgmt" {
			return !containerstatus.Ready
		}
	}

	return podDown
}

// findLastBackupStat will find the status of given backup from last completed-backup
func findLastBackupStat(clientset clientset.Interface, backUp *openebsapis.CStorBackup) openebsapis.CStorBackupStatus {
	lastbkpname := backUp.Spec.BackupName + "-" + backUp.Spec.VolumeName
	lastbkp, err := clientset.OpenebsV1alpha1().CStorCompletedBackups(backUp.Namespace).Get(lastbkpname, metav1.GetOptions{})
	if err != nil {
		// Unable to fetch the last backup, so we will return fail state
		klog.Errorf("Failed to fetch last completed-backup:%s error:%s", lastbkpname, err.Error())
		return openebsapis.BKPCStorStatusFailed
	}

	// lastbkp stores the last(PrevSnapName) and 2nd last(SnapName) completed snapshot
	// let's check if last backup's snapname/PrevSnapName  matches with current snapshot name
	if backUp.Spec.SnapName == lastbkp.Spec.SnapName || backUp.Spec.SnapName == lastbkp.Spec.PrevSnapName {
		return openebsapis.BKPCStorStatusDone
	}

	// lastbackup snap/prevsnap doesn't match with bkp snapname
	return openebsapis.BKPCStorStatusFailed
}

// updateBackupStatus will update the backup status to given status
func updateBackupStatus(clientset clientset.Interface, backUp *openebsapis.CStorBackup, status openebsapis.CStorBackupStatus) {
	backUp.Status = status

	_, err := clientset.OpenebsV1alpha1().CStorBackups(backUp.Namespace).Update(backUp)
	if err != nil {
		klog.Errorf("Failed to update backup:%s with status:%v", backUp.Name, status)
		return
	}
}

// fetchPoolManagerFromCSPI returns pool manager pod for provided CSPI
func fetchPoolManagerFromCSPI(k8sclientset kubernetes.Interface, cspiName string) (*corev1.Pod, error) {
	cstorPodLabel := "app=cstor-pool"
	cspiPoolName := cstortypes.CStorPoolInstanceLabelKey + "=" + cspiName
	podlistops := metav1.ListOptions{
		LabelSelector: cstorPodLabel + "," + cspiPoolName,
	}

	openebsNs := getOpenEBSNamespace()
	if openebsNs == "" {
		return nil, errors.Errorf("Failed to fetch operator namespace")
	}

	podList, err := k8sclientset.CoreV1().Pods(openebsNs).List(podlistops)
	if err != nil {
		klog.Errorf("Failed to fetch pod list :%v", err)
		return nil, err
	}

	if len(podList.Items) != 1 {
		return nil, errors.Errorf("expected 1 pool manager but got %d pool managers", len(podList.Items))
	}
	return &podList.Items[0], nil
}

// TODO: Move below functions into API because there kind of getter methods.

// isBackupCompleted returns true if backup execution is completed
func isBackupCompleted(backUp *openebsapis.CStorBackup) bool {
	if isBackupFailed(backUp) || isBackupSucceeded(backUp) {
		return true
	}
	return false
}

// isBackupFailed returns true if backup failed
func isBackupFailed(backUp *openebsapis.CStorBackup) bool {
	if backUp.Status == openebsapis.BKPCStorStatusFailed {
		return true
	}
	return false
}

// isBackupSucceeded returns true if backup completed successfully
func isBackupSucceeded(backUp *openebsapis.CStorBackup) bool {
	if backUp.Status == openebsapis.BKPCStorStatusDone {
		return true
	}
	return false
}
