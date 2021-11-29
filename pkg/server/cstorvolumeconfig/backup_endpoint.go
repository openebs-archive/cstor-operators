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
	"net/http"

	"encoding/json"
	"fmt"
	"strings"

	version "github.com/hashicorp/go-version"
	cstorapis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	cstortypes "github.com/openebs/api/v3/pkg/apis/types"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	"github.com/openebs/api/v3/pkg/util"
	snapshot "github.com/openebs/cstor-operators/pkg/snapshot"
	"github.com/pkg/errors"
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
	snapshotter  snapshot.Snapshotter
	namespace    string
}

var (
	// openebsNamespace is the namespace where openebs is deployed
	openebsNamespace string
	// minV1SupportedVersion is the minimum OpenEBS version required to perfrom
	// CRUD operations on v1 cStor backup and restore resources
	minV1SupportedVersion *version.Version
)

func init() {
	var err error
	// 2.2.0 is minimum version that supports v1 version of cStor and backup
	minV1SupportedVersion, err = version.NewVersion("2.2.0")
	if err != nil {
		klog.Fatalf("failed to parse 2.2.0 version error: %v", err)
	}
}

/***************************REST ENDPOINTS**********************************************************************************************
 * curl on CVC service with port 5757 then it will create backup related resources. We can use below example to execute POST, GET and DELETE methods
 * POST method curl -XPOST -d '{"metadata":{"namespace":"openebs","creationTimestamp":null},"spec":{"backupName":"testbackup","volumeName":"pvc-185eb80c-f23e-42ea-8136-8863c1c9eb0e","snapName":"backup-snapshot2","prevSnapName":"","backupDest":"172.02.29.12:343132","localSnap":false},"status":""}'  http://10.101.149.30:5757/latest/backups/
 *
 **************************************************************************************************************************************
 * GET method  curl -XGET -d '{"metadata":{"namespace":"openebs","creationTimestamp":null},"spec":{"backupName":"testbackup","volumeName":"pvc-185eb80c-f23e-42ea-8136-8863c1c9eb0e","snapName":"backup-snapshot2","prevSnapName":"","backupDest":"172.02.29.12:343132","localSnap":false},"status":""}'  http://10.101.149.30:5757/latest/backups/
 *
 **************************************************************************************************************************************
 *DELETE method curl -XDELETE http://10.101.149.30:5757/latest/backups/backup-snapshot2?volume=pvc-185eb80c-f23e-42ea-8136-8863c1c9eb0e\&namespace=openebs\&schedule=testbackup
 *
 * Here IP address should be CVC-Operator service IP
 **************************************************************************************************************************************
 */

// backupV1alpha1SpecificRequest deals with backup API requests
func (s *HTTPServer) backupV1alpha1SpecificRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	backupOp := &backupAPIOps{
		req:          req,
		resp:         resp,
		k8sclientset: s.cvcServer.kubeclientset,
		clientset:    s.cvcServer.clientset,
		snapshotter:  s.cvcServer.snapshotter,
		namespace:    getOpenEBSNamespace(),
	}

	switch req.Method {
	case "POST":
		klog.Infof("Got backup POST request")
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
	backup := &cstorapis.CStorBackup{}

	err := decodeBody(bOps.req, backup)
	if err != nil {
		return nil, err
	}

	if err := backupCreateRequestValidations(backup); err != nil {
		return nil, err
	}
	klog.Infof("Requested to create backup for volume %s/%s remoteBackup: %t", backup.Namespace, backup.Spec.VolumeName, !backup.Spec.LocalSnap)

	// TODO: Move this to interface so that we can mock
	// snapshot calls
	snapshot := snapshot.Snapshot{
		VolumeName:   backup.Spec.VolumeName,
		SnapshotName: backup.Spec.SnapName,
		Namespace:    bOps.namespace,
		SnapClient:   bOps.snapshotter,
	}
	snapResp, err := snapshot.CreateSnapshot(bOps.clientset)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to create snapshot '%v'", err))
	}
	klog.Infof("Backup snapshot:'%s' created successfully for volume:%s response: %s", backup.Spec.SnapName, backup.Spec.VolumeName, snapResp)

	// In case of local backup no need to create CStorBackup CR
	if backup.Spec.LocalSnap {
		return "", nil
	}

	backup.Name = backup.Spec.SnapName + "-" + backup.Spec.VolumeName

	// find healthy CVR which will helps to create backup CR
	cvr, err := findHealthyCVR(bOps.clientset, backup.Spec.VolumeName, bOps.namespace)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to find healthy replica for volume %s", backup.Spec.VolumeName))
	}

	poolName := cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey]
	backup.ObjectMeta.Labels = map[string]string{
		cstortypes.CStorPoolInstanceUIDLabelKey:  cvr.ObjectMeta.Labels[cstortypes.CStorPoolInstanceUIDLabelKey],
		cstortypes.CStorPoolInstanceNameLabelKey: poolName,
		cstortypes.PersistentVolumeLabelKey:      cvr.ObjectMeta.Labels[cstortypes.PersistentVolumeLabelKey],
		"openebs.io/backup":                      backup.Spec.BackupName,
	}

	poolVersion, err := getPoolVersion(poolName, bOps.namespace, bOps.clientset)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("failed to get %s pool version error: %v", poolName, err))
	}

	backupInterface, err := bOps.getBackupInterfaceFromPoolVersion(poolVersion, backup)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("failed to get backupInterface error: %v", err))
	}

	lastSnapName, err := backupInterface.getOrCreateLastBackupSnap()
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed get or create lastCompleted backup error: %v", err.Error()))
	}
	// Initialize backup status as pending
	backupInterface.setBackupStatus(string(openebsapis.BKPCStorStatusPending))
	backupInterface.setLastSnapshotName(lastSnapName)

	// NOTE: We are logining by using v1 backup resource irrespective of version
	klog.Infof("Creating backup %s for volume %q poolName: %v poolUUID:%v", backup.Spec.SnapName,
		backup.Spec.VolumeName,
		backup.ObjectMeta.Labels[cstortypes.CStorPoolInstanceNameLabelKey],
		backup.ObjectMeta.Labels[cstortypes.CStorPoolInstanceUIDLabelKey])
	backupInterface, err = backupInterface.createBackupResource()
	if err != nil {
		klog.Errorf("Failed to create backup: error '%s'", err.Error())
		return nil, CodedError(500, err.Error())
	}

	klog.Infof("Backup resource:'%s' created successfully", backup.Name)
	return "", nil
}

// get is http handler which handles backup get request
// It will get the snapshot created by the given backup if backup is done/failed
func (bOps *backupAPIOps) get() (interface{}, error) {
	backup := &cstorapis.CStorBackup{}

	err := decodeBody(bOps.req, backup)
	if err != nil {
		return nil, err
	}

	// backup name is expected
	if len(strings.TrimSpace(backup.Spec.BackupName)) == 0 {
		return nil, CodedError(400, fmt.Sprintf("Failed to get backup: missing backup name "))
	}

	// namespace is expected
	if len(strings.TrimSpace(backup.Namespace)) == 0 {
		return nil, CodedError(400, fmt.Sprintf("Failed to get backup '%v': missing namespace", backup.Spec.BackupName))
	}

	// volume name is expected
	if len(strings.TrimSpace(backup.Spec.VolumeName)) == 0 {
		return nil, CodedError(400, fmt.Sprintf("Failed to get backup '%v': missing volume name", backup.Spec.BackupName))
	}

	backup.Name = backup.Spec.SnapName + "-" + backup.Spec.VolumeName

	backupInterface, err := bOps.getBackupInterface(backup.Name, backup.Namespace)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("failed to get backup interface error: %v", err))
	}

	if !backupInterface.isBackupCompleted() {
		cspiName := backupInterface.getCSPIName()
		// check if node is running or not
		backupNodeDown := checkIfPoolManagerNodeDown(bOps.k8sclientset, cspiName, bOps.namespace)
		// check if cstor-pool-mgmt container is running or not
		backupPodDown := checkIfPoolManagerDown(bOps.k8sclientset, cspiName, bOps.namespace)

		if backupNodeDown || backupPodDown {
			// Backup is stalled, let's find last completed-backup status
			laststat := backupInterface.findLastBackupStat()
			// Update Backup status according to last completed-backup
			backupInterface = backupInterface.updateBackupStatus(laststat)
		}
	}

	out, err := json.Marshal(backupInterface.getBackupObject())
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
func (bOps *backupAPIOps) deleteBackup(snapName, volname, ns, schedule string) error {
	lastCompletedBackup := schedule + "-" + volname
	cstorBackupName := snapName + "-" + volname

	backupInterface, err := bOps.getBackupInterface(cstorBackupName, ns)
	if err != nil {
		return errors.Wrapf(err, "failed to get backup interface")
	}
	// On successfull completion of backup plugin will send delete request to cleanup the backup
	if backupInterface == nil {
		klog.Infof("Backup %s for volume %s already deleted", snapName, volname)
		return nil
	}

	err = backupInterface.deleteCompletedBackup(lastCompletedBackup, ns, snapName)
	if err != nil {
		return errors.Wrapf(err, "failed to delete lastCompletedBackup %s", lastCompletedBackup)
	}

	snapshot := snapshot.Snapshot{
		VolumeName:   volname,
		SnapshotName: snapName,
		Namespace:    bOps.namespace,
		SnapClient:   bOps.snapshotter,
	}
	// Snapshot Name and backup name are same
	_, err = snapshot.DeleteSnapshot(bOps.clientset)
	if err != nil {
		return errors.Wrapf(err, "snapshot deletion failed")
	}

	err = backupInterface.deleteBackup(cstorBackupName, ns)
	if err != nil {
		return errors.Wrapf(err, "failed to delete cstorbackup: %s resource", cstorBackupName)
	}
	return nil
}

// backupCreateRequestValidations validates the backup create request
func backupCreateRequestValidations(backup *cstorapis.CStorBackup) error {
	// backup name is expected
	if len(strings.TrimSpace(backup.Spec.BackupName)) == 0 {
		return CodedError(400, string("Failed to create backup: missing backup name "))
	}

	// namespace is expected
	if len(strings.TrimSpace(backup.Namespace)) == 0 {
		return CodedError(400, fmt.Sprintf("Failed to create backup '%v': missing namespace", backup.Spec.BackupName))
	}

	// volume name is expected
	if len(strings.TrimSpace(backup.Spec.VolumeName)) == 0 {
		return CodedError(400, fmt.Sprintf("Failed to create backup '%v': missing volume name", backup.Spec.BackupName))
	}

	// backupIP is expected for remote snapshot
	if !backup.Spec.LocalSnap && len(strings.TrimSpace(backup.Spec.BackupDest)) == 0 {
		return CodedError(400, fmt.Sprintf("Failed to create backup '%v': missing backupIP", backup.Spec.BackupName))
	}

	// snapshot name is expected
	if len(strings.TrimSpace(backup.Spec.SnapName)) == 0 {
		return CodedError(400, fmt.Sprintf("Failed to create backup '%v': missing snapName", backup.Spec.BackupName))
	}
	return nil
}

// getOpenEBSNamespace returns namespace where
// cvc operator is running
func getOpenEBSNamespace() string {
	if openebsNamespace == "" {
		openebsNamespace = util.GetEnv(util.OpenEBSNamespace)
	}
	return openebsNamespace
}

func (bOps *backupAPIOps) getBackupInterface(backupName,
	backupNamespace string) (backupHelper, error) {

	backupObj, err := bOps.clientset.OpenebsV1alpha1().
		CStorBackups(backupNamespace).
		Get(context.TODO(), backupName, metav1.GetOptions{})
	if err == nil {
		backupInterface := newV1Alpha1BackupWrapper(bOps.clientset).setBackup(backupObj)
		return backupInterface, nil
	}
	if k8serror.IsNotFound(err) {
		backupObj, err := bOps.clientset.CstorV1().
			CStorBackups(backupNamespace).
			Get(context.TODO(), backupName, metav1.GetOptions{})
		if err != nil {
			if !k8serror.IsNotFound(err) {
				return nil, errors.Wrapf(err, "failed to fetch %s backup v1 version also", backupName)
			}
			// This is a case where backup is already deleted
			return nil, nil
		}
		backupInterface := newV1BackupWrapper(bOps.clientset).setBackup(backupObj)
		return backupInterface, nil
	}
	return nil, errors.Wrapf(err, "failed to fetch %s backup", backupName)
}

func (bOps *backupAPIOps) getBackupInterfaceFromPoolVersion(
	poolVersion string, backup *cstorapis.CStorBackup) (backupHelper, error) {
	// error will be reported when version doesn't contain any valid version pattern
	// Ex: dev, ci, master
	// Spliting required if version contains RC1, RC2 to make comparisions
	currentVersion, err := version.NewVersion(strings.Split(poolVersion, "-")[0])
	if err != nil {
		// We need to support backup for ci images also
		if strings.TrimSpace(poolVersion) != "" && !strings.Contains(poolVersion, "dev") {
			return nil, errors.Wrapf(err, "failed to parse %q pool version", poolVersion)
		}
		klog.Warningf("failed to parse pool %s version error: %v will create v1 version of backup and restore", poolVersion, err)
	}

	if currentVersion != nil && currentVersion.LessThan(minV1SupportedVersion) {
		return newV1Alpha1BackupWrapper(bOps.clientset).setBackup(getV1Alpha1BackupFromV1(backup)), nil
	} else if currentVersion != nil && currentVersion.GreaterThanOrEqual(minV1SupportedVersion) {
		return newV1BackupWrapper(bOps.clientset).setBackup(backup), nil
	}
	// return latest supported version v1
	// If code reached over here then pool is in non released version i.e dev/ci
	return newV1BackupWrapper(bOps.clientset).setBackup(backup), nil
}

// findHealthyCVR returns Heathy CVR if exists else
// it will return error
func findHealthyCVR(
	openebsClient clientset.Interface,
	volume, namespace string) (*cstorapis.CStorVolumeReplica, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: cstortypes.PersistentVolumeLabelKey + "=" + volume,
	}

	cvrList, err := openebsClient.CstorV1().CStorVolumeReplicas(namespace).List(context.TODO(), listOptions)
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

func getV1Alpha1BackupFromV1(backup *cstorapis.CStorBackup) *openebsapis.CStorBackup {
	return &openebsapis.CStorBackup{
		ObjectMeta: backup.ObjectMeta,
		Spec: openebsapis.CStorBackupSpec{
			BackupName:   backup.Spec.BackupName,
			VolumeName:   backup.Spec.VolumeName,
			SnapName:     backup.Spec.SnapName,
			PrevSnapName: backup.Spec.PrevSnapName,
			BackupDest:   backup.Spec.BackupDest,
			LocalSnap:    backup.Spec.LocalSnap,
		},
		Status: openebsapis.CStorBackupStatus(backup.Status),
	}
}

func getPoolVersion(cspiName, cspiNamespace string, clientset clientset.Interface) (string, error) {
	cspi, err := clientset.CstorV1().CStorPoolInstances(cspiNamespace).Get(context.TODO(), cspiName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return cspi.VersionDetails.Status.Current, nil
}
