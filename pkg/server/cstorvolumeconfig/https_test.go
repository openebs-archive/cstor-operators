/*
Copyright 2020 The OpenEBS Authors

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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	cstorapis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	cstortypes "github.com/openebs/api/v3/pkg/apis/types"
	"github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"

	openebstypes "github.com/openebs/api/v3/pkg/apis/types"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	openebsFakeClientset "github.com/openebs/api/v3/pkg/client/clientset/versioned/fake"
	"github.com/openebs/api/v3/pkg/util"
	server "github.com/openebs/cstor-operators/pkg/server"
	snapshot "github.com/openebs/cstor-operators/pkg/snapshot/snapshottest"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

// NOTE: This test verifies the backup and restore endpoints.
//       Test case covers both v1 and v1alpha1 version

var (
	namespace = "openebs"
)

// fixture encapsulates fake client sets and client-go testing objects.
// This is useful for mocking the endpoint.
type fixture struct {
	t *testing.T
	// k8sClient is the fake client set for k8s native objects.
	k8sClient *fake.Clientset
	// openebsClient is the fake client set for openebs cr objects.
	openebsClient *openebsFakeClientset.Clientset
}

// newFixture returns a new fixture
func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	return f
}

// SetFakeClient initilizes the fake required clientsets
func (f *fixture) SetFakeClient() {
	// Load kubernetes client set by preloading with k8s objects.
	f.k8sClient = fake.NewSimpleClientset()

	// Load openebs client set by preloading with openebs objects.
	f.openebsClient = openebsFakeClientset.NewSimpleClientset()
}

func (f *fixture) fakeNodeCreator(nodeCount int) {
	for i := 0; i < nodeCount; i++ {
		node := &corev1.Node{}
		node.Name = "worker-" + strconv.Itoa(i)
		labels := make(map[string]string)
		labels["kubernetes.io/hostname"] = node.Name
		node.Labels = labels
		node.Status.Conditions = []corev1.NodeCondition{}
		_, err := f.k8sClient.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
		if err != nil && !k8serror.IsAlreadyExists(err) {
			klog.Error(err)
			continue
		}
		_, err = f.k8sClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
		if err != nil {
			klog.Error(err)
		}
	}
}

func (f *fixture) fakePoolsCreator(cspcName string, poolVersions []string, poolCount int) error {
	nodeList, err := f.k8sClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(nodeList.Items) < poolCount {
		return errors.Errorf("enough nodes doesn't exist to create fake CSPIs")
	}
	var copyPoolVersions []string
	if len(poolVersions) >= poolCount {
		copyPoolVersions = poolVersions
	} else {
		copyPoolVersions = poolVersions
		// Fill remaining with index 0 of poolVersions
		for i := len(copyPoolVersions) - 1; i < poolCount; i++ {
			copyPoolVersions = append(copyPoolVersions, poolVersions[0])
		}
	}

	for i := 0; i < poolCount; i++ {
		labels := map[string]string{
			openebstypes.HostNameLabelKey:         nodeList.Items[i].Name,
			openebstypes.CStorPoolClusterLabelKey: cspcName,
		}
		cspi := cstorapis.NewCStorPoolInstance().
			WithName(cspcName + "-" + rand.String(4)).
			WithNamespace(namespace).
			WithNodeSelectorByReference(nodeList.Items[i].Labels).
			WithNodeName(nodeList.Items[i].Name).
			WithLabelsNew(labels).
			WithNewVersion(copyPoolVersions[i])
		cspi.Status.Phase = cstorapis.CStorPoolStatusOnline
		cspiObj, err := f.openebsClient.CstorV1().CStorPoolInstances(namespace).Create(context.TODO(), cspi, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to create fake cspi")
		}
		err = f.createFakePoolPod(cspiObj)
		if err != nil {
			return errors.Wrapf(err, "failed to create fake pool pod")
		}
	}
	return nil
}

// createFakePoolPod creates the fake CStorPool pod for given based on provided CSPI
func (f *fixture) createFakePoolPod(cspi *cstorapis.CStorPoolInstance) error {
	// NOTE: Filling only required information
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cspi.Name + "-" + rand.String(6),
			Namespace: namespace,
			Labels: map[string]string{
				"app":                                  "cstor-pool",
				openebstypes.CStorPoolInstanceLabelKey: cspi.Name,
			},
		},
		Spec: corev1.PodSpec{
			NodeName: cspi.Spec.HostName,
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "cstor-pool-mgmt",
					Ready: true,
				},
				{
					Name:  "cstor-pool",
					Ready: true,
				},
			},
		},
	}
	_, err := f.k8sClient.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	return err
}

// createFakeVolumeReplicas will create fake CVRs on cstorPools
func (f *fixture) createFakeVolumeReplicas(
	cspcName, volumeName string, replicaCount int, phase cstorapis.CStorVolumeReplicaPhase) error {
	cspiList, err := f.openebsClient.CstorV1().CStorPoolInstances(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: openebstypes.CStorPoolClusterLabelKey + "=" + cspcName,
	})
	if err != nil {
		return err
	}
	if len(cspiList.Items) < replicaCount {
		return errors.Errorf("enough pools doesn't exist to create fake CVRs")
	}
	for i := 0; i < replicaCount; i++ {
		labels := map[string]string{
			openebstypes.CStorPoolInstanceNameLabelKey: cspiList.Items[i].Name,
			openebstypes.PersistentVolumeLabelKey:      volumeName,
			"cstorvolume.openebs.io/name":              volumeName,
		}
		cvr := cstorapis.NewCStorVolumeReplica().
			WithName(volumeName + "-" + cspiList.Items[i].Name).
			WithLabelsNew(labels).
			WithStatusPhase(phase)
		_, err := f.openebsClient.CstorV1().CStorVolumeReplicas(namespace).Create(context.TODO(), cvr, metav1.CreateOptions{})
		if err != nil && !k8serror.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

// createFakeCStorVolume will create fake CStorVolume
func (f *fixture) createFakeCStorVolume(volumeName string) error {
	labels := map[string]string{
		openebstypes.PersistentVolumeLabelKey: volumeName,
	}
	cv := cstorapis.NewCStorVolume().
		WithNamespace(namespace).
		WithName(volumeName).
		WithLabelsNew(labels)
	_, err := f.openebsClient.CstorV1().CStorVolumes(namespace).Create(context.TODO(), cv, metav1.CreateOptions{})
	if err != nil && !k8serror.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (f *fixture) createCStorCompletedBackup(backup *openebsapis.CStorBackup, previousSnapshot string) error {
	lastbkpName := backup.Spec.BackupName + "-" + backup.Spec.VolumeName
	// Build CStorCompletedBackup which will helpful for incremental backups
	bk := &openebsapis.CStorCompletedBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lastbkpName,
			Namespace: backup.Namespace,
			Labels:    backup.Labels,
		},
		Spec: openebsapis.CStorBackupSpec{
			BackupName:   backup.Spec.BackupName,
			VolumeName:   backup.Spec.VolumeName,
			PrevSnapName: previousSnapshot,
		},
	}
	_, err := f.openebsClient.OpenebsV1alpha1().CStorCompletedBackups(bk.Namespace).Create(context.TODO(), bk, metav1.CreateOptions{})
	return err
}

func (f *fixture) fakeCVCRoutine(channel chan int) {
	fmt.Printf("Fake CVC routine has started")
	channel <- 1
	for {
		cvcList, err := f.openebsClient.CstorV1().CStorVolumeConfigs(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			klog.Error(err)
		}

		for _, cvcObj := range cvcList.Items {
			if cvcObj.Annotations[openebstypes.OpenEBSDisableReconcileLabelKey] == "true" {
				klog.Infof("Skipping Reconcilation for CVC %s", cvcObj.Name)
				continue
			}
			if cvcObj.Status.Phase == cstorapis.CStorVolumeConfigPhasePending {
				cspcName := cvcObj.Labels[string(openebstypes.CStorPoolClusterLabelKey)]
				err := f.createFakeVolumeReplicas(cspcName, cvcObj.Name, cvcObj.Spec.Provision.ReplicaCount, cstorapis.CVRStatusOnline)
				if err != nil {
					klog.Error(err)
					time.Sleep(2 * time.Second)
					continue
				}
				err = f.createFakeCStorVolume(cvcObj.Name)
				if err != nil {
					klog.Error(err)
					time.Sleep(2 * time.Second)
					continue
				}
				cvcObj.Status.Phase = cstorapis.CStorVolumeConfigPhaseBound
				_, err = f.openebsClient.CstorV1().CStorVolumeConfigs(namespace).Update(context.TODO(), &cvcObj, metav1.UpdateOptions{})
				if err != nil {
					klog.Error(err)
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func executeCreateBackup(
	httpServer *HTTPServer,
	backupObj *openebsapis.CStorBackup) (http.HandlerFunc, *http.Request, error) {
	//Marshal serializes the value provided into a json document
	jsonValue, _ := json.Marshal(backupObj)

	// Create a request to pass to handler
	req, _ := http.NewRequest("POST", "/latest/backups/", bytes.NewBuffer(jsonValue))
	// create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	req.Header.Add("Content-Type", "application/json")
	handler := http.HandlerFunc(httpServer.wrap(httpServer.backupV1alpha1SpecificRequest))
	handler.ServeHTTP(rr, req)
	// Verify all the required results
	if rr.Code != http.StatusOK {
		data, _ := ioutil.ReadAll(rr.Body)
		return nil, nil, errors.Errorf("failed to create backup for volume %s return code %d error: %s",
			backupObj.Spec.VolumeName, rr.Code, string(data))
	}
	return handler, req, nil
}

func verifyExistenceOfPendingV1Alpha1Backup(name, namespace string, openebsClient clientset.Interface) error {
	backUp, err := openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if backUp.Status != openebsapis.BKPCStorStatusPending {
		return errors.Errorf("expected %s status but got %s on CStorBackup %s",
			openebsapis.BKPCStorStatusPending,
			backUp.Status,
			name,
		)
	}
	return nil
}

func verifyExistenceOfPendingV1Backup(name, namespace string, openebsClient clientset.Interface) error {
	backUp, err := openebsClient.CstorV1().CStorBackups(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if backUp.Status != cstorapis.BKPCStorStatusPending {
		return errors.Errorf("expected %s status but got %s on CStorBackup %s",
			openebsapis.BKPCStorStatusPending,
			backUp.Status,
			name,
		)
	}
	return nil
}

func backupShouldNotExist(name, namespace string, openebsClient clientset.Interface) error {
	_, err := openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if k8serror.IsNotFound(err) {
		return nil
	}
	if err == nil {
		return errors.Errorf("expected for %s not to exist", name)
	}
	return err
}

func TestBackupPostEndPoint(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	f.fakeNodeCreator(5)
	tests := map[string]struct {
		// cspcName used to create fake cstor pools
		cspcName    string
		poolVersion string
		// cstorBackup used to query on backup endpoint
		cstorBackup *openebsapis.CStorBackup
		// snapshotter is used to mock snapshot operations on volumes
		snapshotter *snapshot.FakeSnapshotter
		// cvrStatus creates CVR with provided phase
		cvrStatus                            cstorapis.CStorVolumeReplicaPhase
		isScheduledBackup                    bool
		expectedResponseCode                 int
		verifyBackUpStatus                   func(name, namespace string, openebsClient clientset.Interface) error
		checkExistenceOfCStorCompletedBackup bool
		isV1Version                          bool
	}{
		"When all the resources exist and trigered backup endpoint with post method": {
			cspcName:    "cspc-disk-pool1",
			poolVersion: "1.10.0",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup1",
					VolumeName: "volume1",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot1",
				},
			},
			snapshotter:                          &snapshot.FakeSnapshotter{},
			cvrStatus:                            cstorapis.CVRStatusOnline,
			expectedResponseCode:                 http.StatusOK,
			verifyBackUpStatus:                   verifyExistenceOfPendingV1Alpha1Backup,
			checkExistenceOfCStorCompletedBackup: true,
		},
		"When creation of snapshot fails": {
			cspcName:    "cspc-disk-pool2",
			poolVersion: "1.11.0",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup2",
					VolumeName: "volume2",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot",
				},
			},
			snapshotter: &snapshot.FakeSnapshotter{
				ShouldReturnFakeError: true,
			},
			expectedResponseCode: http.StatusBadRequest,
			verifyBackUpStatus:   backupShouldNotExist,
			cvrStatus:            cstorapis.CVRStatusOnline,
		},
		"When CStorBackup doesn't have snapname": {
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup2",
					VolumeName: "volume2",
					BackupDest: "172.102.29.12:3234",
				},
			},
			snapshotter: &snapshot.FakeSnapshotter{
				ShouldReturnFakeError: true,
			},
			expectedResponseCode: http.StatusBadRequest,
			verifyBackUpStatus:   backupShouldNotExist,
			cvrStatus:            cstorapis.CVRStatusOnline,
		},
		"When cvrs are not healthy backup should fail": {
			cspcName:    "cspc-disk-pool3",
			poolVersion: "1.12.3",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup3",
					VolumeName: "volume3",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot",
				},
			},
			snapshotter:          &snapshot.FakeSnapshotter{},
			expectedResponseCode: http.StatusBadRequest,
			verifyBackUpStatus:   backupShouldNotExist,
			cvrStatus:            cstorapis.CVRStatusOffline,
		},
		"When request is to create scheduled backup": {
			cspcName:    "cspc-disk-pool4",
			poolVersion: "2.0.0",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup4",
					VolumeName: "volume4",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot4",
				},
			},
			snapshotter:          &snapshot.FakeSnapshotter{},
			expectedResponseCode: http.StatusOK,
			verifyBackUpStatus:   verifyExistenceOfPendingV1Alpha1Backup,
			cvrStatus:            cstorapis.CVRStatusOnline,
			isScheduledBackup:    true,
		},
		"When local backup is requested": {
			cspcName:    "cspc-disk-pool5",
			poolVersion: "2.0.4",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup-local1",
					VolumeName: "volume-local1",
					SnapName:   "snapshot-local1",
					LocalSnap:  true,
				},
			},
			snapshotter:          &snapshot.FakeSnapshotter{},
			expectedResponseCode: http.StatusOK,
			verifyBackUpStatus:   backupShouldNotExist,
			cvrStatus:            cstorapis.CVRStatusOffline,
		},
		"When all the resources exist and trigered backup endpoint with post method when pool supports v1 version": {
			cspcName:    "cspc-disk-pool6",
			poolVersion: "2.5.0",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup6",
					VolumeName: "volume6",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot6",
				},
			},
			snapshotter:                          &snapshot.FakeSnapshotter{},
			cvrStatus:                            cstorapis.CVRStatusOnline,
			expectedResponseCode:                 http.StatusOK,
			verifyBackUpStatus:                   verifyExistenceOfPendingV1Backup,
			checkExistenceOfCStorCompletedBackup: true,
			isV1Version:                          true,
		},
		"test backup when pool is in RC versions": {
			cspcName:    "cspc-disk-pool7",
			poolVersion: "2.2.0-RC2",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup7",
					VolumeName: "volume7",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot7",
				},
			},
			snapshotter:                          &snapshot.FakeSnapshotter{},
			cvrStatus:                            cstorapis.CVRStatusOnline,
			expectedResponseCode:                 http.StatusOK,
			verifyBackUpStatus:                   verifyExistenceOfPendingV1Backup,
			checkExistenceOfCStorCompletedBackup: true,
			isV1Version:                          true,
		},
	}
	os.Setenv(util.OpenEBSNamespace, "openebs")
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			prevSnapshot := "previous-snapshot"
			backupName := test.cstorBackup.Spec.SnapName + "-" + test.cstorBackup.Spec.VolumeName
			lastbkpName := test.cstorBackup.Spec.BackupName + "-" + test.cstorBackup.Spec.VolumeName
			f.fakePoolsCreator(test.cspcName, []string{test.poolVersion}, 5)
			f.createFakeVolumeReplicas(test.cspcName, test.cstorBackup.Spec.VolumeName, 3, test.cvrStatus)
			f.createFakeCStorVolume(test.cstorBackup.Spec.VolumeName)
			// If schedule backup is set then create CStorCompletedBackup(on assumption it is other than first request)
			if test.isScheduledBackup {
				f.createCStorCompletedBackup(test.cstorBackup, prevSnapshot)
			}
			// Create HTTPServer with all the required clients
			httpServer := &HTTPServer{
				cvcServer: NewCVCServer(server.DefaultServerConfig(), os.Stdout).
					WithOpenebsClientSet(f.openebsClient).
					WithKubernetesClientSet(f.k8sClient).
					WithSnapshotter(test.snapshotter),
				logger: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
			}
			//Marshal serializes the value provided into a json document
			jsonValue, err := json.Marshal(test.cstorBackup)
			if err != nil {
				t.Errorf("failed to marshal cstor backup error: %s", err.Error())
			}
			// Create a request to pass to our handler
			req, err := http.NewRequest("POST", "/latest/backups/", bytes.NewBuffer(jsonValue))
			if err != nil {
				t.Errorf("failed to build request error: %s", err.Error())
			}
			// create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			rr := httptest.NewRecorder()
			req.Header.Add("Content-Type", "application/json")
			handler := http.HandlerFunc(httpServer.wrap(httpServer.backupV1alpha1SpecificRequest))
			handler.ServeHTTP(rr, req)
			// Verify all the required results
			if rr.Code != test.expectedResponseCode {
				data, err := ioutil.ReadAll(rr.Body)
				if err != nil {
					t.Errorf("Unable to read response from server %v", err)
				}
				t.Errorf("handler returned wrong status code: got %v want %v response %s", rr.Code, test.expectedResponseCode, string(data))
			}
			if test.verifyBackUpStatus != nil {
				if err := test.verifyBackUpStatus(backupName, namespace, f.openebsClient); err != nil {
					t.Errorf("failed to verify backup status failed error: %s", err.Error())
				}
			}
			if test.checkExistenceOfCStorCompletedBackup {
				if test.isV1Version {
					_, err = f.openebsClient.CstorV1().CStorCompletedBackups(namespace).Get(context.TODO(), lastbkpName, metav1.GetOptions{})
				} else {
					_, err = f.openebsClient.OpenebsV1alpha1().CStorCompletedBackups(namespace).Get(context.TODO(), lastbkpName, metav1.GetOptions{})
				}
				if err != nil {
					t.Errorf("failed to verify cstorcompleted backup existence error: %s", err.Error())
				}
			}
			if test.isScheduledBackup {
				backupObj, err := f.openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(context.TODO(), backupName, metav1.GetOptions{})
				if err != nil {
					t.Errorf("failed to get details of backup error: %v", err)
				}
				if backupObj.Spec.PrevSnapName != prevSnapshot {
					t.Errorf("%q test failed want previous snapshot name %q but got %q", name, prevSnapshot, backupObj.Spec.PrevSnapName)
				}
			}
		})
	}
	os.Unsetenv(util.OpenEBSNamespace)
}

func TestBackupGetEndPoint(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	tests := map[string]struct {
		// cspcName used to create fake cstor pools
		cspcName string
		// cstorBackup used to query on backup endpoint
		cstorBackup               *openebsapis.CStorBackup
		shouldMarkPoolPodDown     bool
		shouldMarkPoolPodNodeDown bool
		expectedBackupStatus      openebsapis.CStorBackupStatus
	}{
		"When backup is triggered and expecting status to be pending": {
			cspcName: "cspc-cstor-pool1",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup1",
					VolumeName: "volume1",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot1",
				},
			},
			expectedBackupStatus: openebsapis.BKPCStorStatusPending,
		},
		"While fetching the status if pool manager is down": {
			cspcName: "cspc-cstor-pool2",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup2",
					VolumeName: "volume2",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot2",
				},
			},
			shouldMarkPoolPodDown: true,
			expectedBackupStatus:  openebsapis.BKPCStorStatusFailed,
		},
		"While fetching the status if pool manager node is down": {
			cspcName: "cspc-cstor-pool3",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup3",
					VolumeName: "volume3",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot3",
				},
			},
			shouldMarkPoolPodNodeDown: true,
			expectedBackupStatus:      openebsapis.BKPCStorStatusFailed,
		},
	}
	os.Setenv(util.OpenEBSNamespace, "openebs")
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			backupName := test.cstorBackup.Spec.SnapName + "-" + test.cstorBackup.Spec.VolumeName
			f.fakeNodeCreator(5)
			f.fakePoolsCreator(test.cspcName, []string{"2.0.0"}, 5)
			f.createFakeVolumeReplicas(test.cspcName, test.cstorBackup.Spec.VolumeName, 3, cstorapis.CVRStatusOnline)
			f.createFakeCStorVolume(test.cstorBackup.Spec.VolumeName)
			// ================================= Creating Backup Using ENDPOINT ======================================
			// Create HTTPServer
			httpServer := &HTTPServer{
				cvcServer: NewCVCServer(server.DefaultServerConfig(), os.Stdout).
					WithOpenebsClientSet(f.openebsClient).
					WithKubernetesClientSet(f.k8sClient).
					WithSnapshotter(&snapshot.FakeSnapshotter{}),
				logger: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
			}
			handler, req, err := executeCreateBackup(httpServer, test.cstorBackup)
			// Verify all the required results
			if err != nil {
				t.Errorf("error: %v", err)
			}

			// ================================= Change according to the test configurations =========================
			backupObj, err := f.openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(context.TODO(), backupName, metav1.GetOptions{})
			if err == nil && test.shouldMarkPoolPodDown {
				cspiName := backupObj.Labels[openebstypes.CStorPoolInstanceNameLabelKey]
				poolManager, err := getPoolManager(f.k8sClient, cspiName, namespace)
				if err == nil {
					for i, containerstatus := range poolManager.Status.ContainerStatuses {
						if containerstatus.Name == "cstor-pool-mgmt" {
							poolManager.Status.ContainerStatuses[i].Ready = false
						}
						f.k8sClient.CoreV1().Pods(namespace).Update(context.TODO(), poolManager, metav1.UpdateOptions{})
					}

				}
			}
			if err == nil && test.shouldMarkPoolPodNodeDown {
				cspiName := backupObj.Labels[openebstypes.CStorPoolInstanceNameLabelKey]
				poolManager, err := getPoolManager(f.k8sClient, cspiName, namespace)
				if err == nil {
					node, err := f.k8sClient.CoreV1().Nodes().Get(context.TODO(), poolManager.Spec.NodeName, metav1.GetOptions{})
					if err == nil {
						node.Status.Conditions = append(node.Status.Conditions, corev1.NodeCondition{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionFalse,
						})
						f.k8sClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
					}
				}
			}

			// ==================== Build and Make REST request ========================
			//Marshal serializes the value provided into a json document
			jsonValue, _ := json.Marshal(backupObj)
			rr := httptest.NewRecorder()
			req.Header.Add("Content-Type", "application/json")
			// Create a request to pass to handler
			req, err = http.NewRequest("GET", "/latest/backups/", bytes.NewBuffer(jsonValue))
			if err != nil {
				t.Errorf("failed to build request error: %s", err.Error())
			}
			handler.ServeHTTP(rr, req)
			// Verify all the required results
			if rr.Code != http.StatusOK {
				data, _ := ioutil.ReadAll(rr.Body)
				t.Errorf("failed to backup return code %d error: %s", rr.Code, string(data))
			}

			backupObj, err = f.openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(context.TODO(), backupName, metav1.GetOptions{})
			if err != nil {
				t.Errorf("failed to get details of backup error: %v", err)
			}
			if backupObj.Status != test.expectedBackupStatus {
				t.Errorf("%q test failed want expected status %q but got %q", name, test.expectedBackupStatus, backupObj.Status)
			}
		})
	}
	os.Unsetenv(util.OpenEBSNamespace)
}

func TestBackupDeleteEndPoint(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	tests := map[string]struct {
		// cspcName used to create fake cstor pools
		cspcName    string
		poolVersion string
		// cstorBackup used to query on backup endpoint
		cstorBackup                 *openebsapis.CStorBackup
		cstorCompletedBackup        *openebsapis.CStorCompletedBackup
		cstorCompletedBackupV1      *cstorapis.CStorCompletedBackup
		shouldBackupExists          bool
		shouldCompletedBackupExists bool
		shouldInjectSnapshotError   bool
		shouldInjectPayloadError    bool
		expectedResponseCode        int
		isV1Version                 bool
	}{
		"When delete method is triggered on backup endpoint without any error injection": {
			cspcName:    "cspc-cstor-pool1",
			poolVersion: "1.12.0",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup1",
					VolumeName: "volume1",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot1",
				},
			},
			shouldBackupExists:          false,
			shouldCompletedBackupExists: false,
			expectedResponseCode:        http.StatusOK,
		},
		"When delete method is triggered on backup endpoint by injecting error": {
			cspcName:    "cspc-cstor-pool2",
			poolVersion: "2.0.0",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup2",
					VolumeName: "volume2",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot2",
				},
			},
			shouldBackupExists:          true,
			shouldCompletedBackupExists: false,
			shouldInjectSnapshotError:   true,
			expectedResponseCode:        http.StatusInternalServerError,
		},
		"Should not delete cstorcompleted backup": {
			cspcName:    "cspc-cstor-pool3",
			poolVersion: "2.0.0",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup3",
					VolumeName: "volume3",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot3",
				},
			},
			cstorCompletedBackup: &openebsapis.CStorCompletedBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "backup3" + "-" + "volume3",
				},
				Spec: openebsapis.CStorBackupSpec{
					PrevSnapName: "snapshot2",
					SnapName:     "snapshot1",
				},
			},
			shouldBackupExists:          false,
			shouldCompletedBackupExists: true,
			shouldInjectSnapshotError:   false,
			expectedResponseCode:        http.StatusOK,
		},
		"When required payload is not passed": {
			cspcName:    "cspc-cstor-pool4",
			poolVersion: "2.0.0",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup4",
					VolumeName: "volume4",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot4",
				},
			},
			shouldInjectPayloadError:    true,
			shouldBackupExists:          true,
			shouldCompletedBackupExists: true,
			shouldInjectSnapshotError:   false,
			expectedResponseCode:        http.StatusMethodNotAllowed,
		},
		"When delete method is triggered on v1 backup endpoint without any error injection": {
			cspcName:    "cspc-cstor-pool5",
			poolVersion: "2.2.0",
			// Since json tags are same for v1 and v1alpha version there won't be any problem
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup5",
					VolumeName: "volume5",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot5",
				},
			},
			shouldBackupExists:          false,
			shouldCompletedBackupExists: false,
			expectedResponseCode:        http.StatusOK,
			isV1Version:                 true,
		},
		"Should not delete v1 version of cstorcompleted backup": {
			cspcName: "cspc-cstor-pool6",
			// Passing empty version so it will treat as ci
			poolVersion: "",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup6",
					VolumeName: "volume6",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot6",
				},
			},
			cstorCompletedBackupV1: &cstorapis.CStorCompletedBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "backup6" + "-" + "volume6",
				},
				Spec: cstorapis.CStorCompletedBackupSpec{
					SecondLastSnapName: "snapshot2",
					LastSnapName:       "snapshot1",
				},
			},
			shouldBackupExists:          false,
			shouldCompletedBackupExists: true,
			shouldInjectSnapshotError:   false,
			expectedResponseCode:        http.StatusOK,
			isV1Version:                 true,
		},
		"When delete method is triggered on V1 version of backup endpoint by injecting error": {
			cspcName: "cspc-cstor-pool7",
			// pasing dev version
			poolVersion: "master-dev",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup7",
					VolumeName: "volume7",
					BackupDest: "172.102.29.12:3234",
					SnapName:   "snapshot7",
				},
			},
			shouldBackupExists:          true,
			shouldCompletedBackupExists: false,
			shouldInjectSnapshotError:   true,
			expectedResponseCode:        http.StatusInternalServerError,
			isV1Version:                 true,
		},
	}
	os.Setenv(util.OpenEBSNamespace, "openebs")
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			backupName := test.cstorBackup.Spec.SnapName + "-" + test.cstorBackup.Spec.VolumeName
			lastCompletedBackup := test.cstorBackup.Spec.BackupName + "-" + test.cstorBackup.Spec.VolumeName
			f.fakeNodeCreator(5)
			f.fakePoolsCreator(test.cspcName, []string{test.poolVersion}, 5)
			f.createFakeVolumeReplicas(test.cspcName, test.cstorBackup.Spec.VolumeName, 3, cstorapis.CVRStatusOnline)
			f.createFakeCStorVolume(test.cstorBackup.Spec.VolumeName)
			if test.cstorCompletedBackup != nil {
				_, err := f.openebsClient.OpenebsV1alpha1().CStorCompletedBackups(namespace).Create(context.TODO(), test.cstorCompletedBackup, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("failed to create completed backup error: %v", err)
				}
			}
			if test.cstorCompletedBackupV1 != nil {
				_, err := f.openebsClient.CstorV1().CStorCompletedBackups(namespace).Create(context.TODO(), test.cstorCompletedBackupV1, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("failed to create completed backup error: %v", err)
				}
			}
			// ================================= Creating Backup Using ENDPOINT ======================================
			// Create HTTPServer
			httpServer := &HTTPServer{
				cvcServer: NewCVCServer(server.DefaultServerConfig(), os.Stdout).
					WithOpenebsClientSet(f.openebsClient).
					WithKubernetesClientSet(f.k8sClient).
					WithSnapshotter(&snapshot.FakeSnapshotter{}),
				logger: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
			}
			handler, req, err := executeCreateBackup(httpServer, test.cstorBackup)
			// Verify all the required results
			if err != nil {
				t.Errorf("error: %v", err)
			}
			if test.shouldInjectSnapshotError {
				httpServer.cvcServer = httpServer.cvcServer.WithSnapshotter(&snapshot.FakeSnapshotter{ShouldReturnFakeError: true})
			}
			if test.shouldInjectPayloadError {
				// Over write the existing payload with empty
				test.cstorBackup.Spec.BackupName = ""
			}

			// Delete Backup
			req, err = http.NewRequest("DELETE", "/latest/backups/"+test.cstorBackup.Spec.SnapName, nil)
			if err != nil {
				t.Errorf("failed to create new HTTP request error: %v", err)
			}
			q := req.URL.Query()
			q.Add("volume", test.cstorBackup.Spec.VolumeName)
			q.Add("namespace", test.cstorBackup.Namespace)
			q.Add("schedule", test.cstorBackup.Spec.BackupName)
			req.URL.RawQuery = q.Encode()

			rr := httptest.NewRecorder()
			req.Header.Add("Content-Type", "application/json")
			handler = http.HandlerFunc(httpServer.wrap(httpServer.backupV1alpha1SpecificRequest))
			handler.ServeHTTP(rr, req)
			if rr.Code != test.expectedResponseCode {
				data, _ := ioutil.ReadAll(rr.Body)
				t.Errorf("failed to delete backup for volume %s expected code %d but got %d error: %s",
					test.cstorBackup.Spec.VolumeName, test.expectedResponseCode, rr.Code, string(data))
			}

			// Verifying CStorBackup based on flag
			if test.isV1Version {
				_, err = f.openebsClient.CstorV1().CStorBackups(namespace).Get(context.TODO(), backupName, metav1.GetOptions{})
			} else {
				_, err = f.openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(context.TODO(), backupName, metav1.GetOptions{})
			}
			if err != nil {
				if !k8serror.IsNotFound(err) {
					t.Errorf("failed to get backup of volume %s error %v", test.cstorBackup.Spec.VolumeName, err)
				}
				if test.shouldBackupExists {
					t.Errorf("expected backup to exist but not found")
				}
			}
			if !test.shouldBackupExists && err == nil {
				t.Errorf("expected backup not to exist but found %s", backupName)
			}

			// Verifying CStorCompletedBackup based on version flag
			if test.isV1Version {
				_, err = f.openebsClient.CstorV1().
					CStorCompletedBackups(namespace).
					Get(context.TODO(), lastCompletedBackup, metav1.GetOptions{})
			} else {
				_, err = f.openebsClient.OpenebsV1alpha1().
					CStorCompletedBackups(namespace).
					Get(context.TODO(), lastCompletedBackup, metav1.GetOptions{})
			}
			if err != nil {
				if !k8serror.IsNotFound(err) {
					t.Errorf("failed to get completed backup for volume %s error %v", test.cstorBackup.Spec.VolumeName, err)
				}
				if test.shouldCompletedBackupExists {
					t.Errorf("expected completedbackup to exist but not found")
				}
			}
			if !test.shouldCompletedBackupExists && err == nil {
				t.Errorf("expected completed backup not to exist but found %s", backupName)
			}

		})
	}
}

func TestRestoreEndPoint(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	tests := map[string]struct {
		// cspcName used to create fake cstor pools
		cspcName    string
		poolVersion string
		// cstorRestore is used to create restore
		cstorRestore            *openebsapis.CStorRestore
		cvcObj                  *cstorapis.CStorVolumeConfig
		storageClass            *storagev1.StorageClass
		verifyCStorRestoreCount bool
		expectedResponseCode    int
		isV1Version             bool
	}{
		"When restore is triggered": {
			cspcName:    "cspc-cstor-pool1",
			poolVersion: "2.0.0",
			cstorRestore: &openebsapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorRestoreSpec{
					RestoreName: "restore1",
					VolumeName:  "volume1",
					RestoreSrc:  "127.0.0.1:3422",
				},
			},
			cvcObj: &cstorapis.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "volume1",
					Namespace: namespace,
					Labels: map[string]string{
						openebstypes.CStorPoolClusterLabelKey: "cspc-cstor-pool1",
					},
				},
				Spec: cstorapis.CStorVolumeConfigSpec{
					Provision: cstorapis.VolumeProvision{
						ReplicaCount: 3,
					},
				},
				Status: cstorapis.CStorVolumeConfigStatus{
					Phase: cstorapis.CStorVolumeConfigPhasePending,
				},
			},
			expectedResponseCode:    http.StatusOK,
			verifyCStorRestoreCount: true,
		},
		"When restore is triggered but CVC is not marked as bound": {
			cspcName:    "cspc-cstor-pool2",
			poolVersion: "1.12.0",
			cstorRestore: &openebsapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorRestoreSpec{
					RestoreName: "restore2",
					VolumeName:  "volume2",
					RestoreSrc:  "127.0.0.1:3432",
				},
			},
			cvcObj: &cstorapis.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "volume2",
					Namespace: namespace,
					Labels: map[string]string{
						openebstypes.CStorPoolClusterLabelKey: "cspc-cstor-pool2",
					},
					Annotations: map[string]string{
						openebstypes.OpenEBSDisableReconcileLabelKey: "true",
					},
				},
				Spec: cstorapis.CStorVolumeConfigSpec{
					Provision: cstorapis.VolumeProvision{
						ReplicaCount: 3,
					},
				},
				Status: cstorapis.CStorVolumeConfigStatus{
					Phase: cstorapis.CStorVolumeConfigPhasePending,
				},
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		"When restore source name is empty": {
			cspcName:    "cspc-cstor-pool3",
			poolVersion: "1.12.1",
			cstorRestore: &openebsapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorRestoreSpec{
					RestoreName: "restore3",
					VolumeName:  "volume3",
				},
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		"When local restore is triggered": {
			cspcName:    "cspc-cstor-pool4",
			poolVersion: "2.0.0",
			cstorRestore: &openebsapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorRestoreSpec{
					RestoreName:  "restore4",
					VolumeName:   "volume4",
					RestoreSrc:   "snapshot4",
					Local:        true,
					StorageClass: "storage-class4",
					Size:         resource.MustParse("5G"),
				},
			},
			expectedResponseCode: http.StatusOK,
			storageClass: &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "storage-class4",
				},
				Parameters: map[string]string{
					"cstorPoolCluster": "cspc-cstor-pool4",
					"replicaCount":     "3",
				},
			},
		},
		"When local restore is triggered without creating storageclass": {
			cspcName:    "cspc-cstor-pool5",
			poolVersion: "1.11.0",
			cstorRestore: &openebsapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorRestoreSpec{
					RestoreName:  "restore5",
					VolumeName:   "volume5",
					RestoreSrc:   "snapshot5",
					Local:        true,
					StorageClass: "storage-class5",
					Size:         resource.MustParse("5G"),
				},
			},
			expectedResponseCode: http.StatusOK,
			storageClass: &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "storage-class5",
				},
				Parameters: map[string]string{
					"cstorPoolCluster": "cspc-cstor-pool5",
					"replicaCount":     "3",
				},
			},
		},
		"When local restore is triggered without passing storageclass": {
			cspcName:    "cspc-cstor-pool6",
			poolVersion: "2.0.0",
			cstorRestore: &openebsapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorRestoreSpec{
					RestoreName: "restore6",
					VolumeName:  "volume6",
					RestoreSrc:  "snapshot6",
					Local:       true,
					Size:        resource.MustParse("5G"),
				},
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		"When restore is triggered for pools on v1 version": {
			cspcName:    "cspc-cstor-pool7",
			poolVersion: "2.2.0",
			cstorRestore: &openebsapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorRestoreSpec{
					RestoreName: "restore7",
					VolumeName:  "volume7",
					RestoreSrc:  "127.0.0.1:3422",
				},
			},
			cvcObj: &cstorapis.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "volume7",
					Namespace: namespace,
					Labels: map[string]string{
						openebstypes.CStorPoolClusterLabelKey: "cspc-cstor-pool7",
					},
				},
				Spec: cstorapis.CStorVolumeConfigSpec{
					Provision: cstorapis.VolumeProvision{
						ReplicaCount: 3,
					},
				},
				Status: cstorapis.CStorVolumeConfigStatus{
					Phase: cstorapis.CStorVolumeConfigPhasePending,
				},
			},
			expectedResponseCode:    http.StatusOK,
			verifyCStorRestoreCount: true,
			isV1Version:             true,
		},
		"When restore is triggered for pools on v1 version with dev": {
			cspcName:    "cspc-cstor-pool8",
			poolVersion: "master-dev",
			cstorRestore: &openebsapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorRestoreSpec{
					RestoreName: "restore8",
					VolumeName:  "volume8",
					RestoreSrc:  "127.0.0.1:3422",
				},
			},
			cvcObj: &cstorapis.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "volume8",
					Namespace: namespace,
					Labels: map[string]string{
						openebstypes.CStorPoolClusterLabelKey: "cspc-cstor-pool8",
					},
				},
				Spec: cstorapis.CStorVolumeConfigSpec{
					Provision: cstorapis.VolumeProvision{
						ReplicaCount: 3,
					},
				},
				Status: cstorapis.CStorVolumeConfigStatus{
					Phase: cstorapis.CStorVolumeConfigPhasePending,
				},
			},
			expectedResponseCode:    http.StatusOK,
			verifyCStorRestoreCount: true,
			isV1Version:             true,
		},
		"When restore is triggered for pools on v1 supported RC version": {
			cspcName:    "cspc-cstor-pool9",
			poolVersion: "2.2.0-RC3",
			cstorRestore: &openebsapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorRestoreSpec{
					RestoreName: "restore9",
					VolumeName:  "volume9",
					RestoreSrc:  "127.0.0.1:3422",
				},
			},
			cvcObj: &cstorapis.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "volume9",
					Namespace: namespace,
					Labels: map[string]string{
						openebstypes.CStorPoolClusterLabelKey: "cspc-cstor-pool9",
					},
				},
				Spec: cstorapis.CStorVolumeConfigSpec{
					Provision: cstorapis.VolumeProvision{
						ReplicaCount: 3,
					},
				},
				Status: cstorapis.CStorVolumeConfigStatus{
					Phase: cstorapis.CStorVolumeConfigPhasePending,
				},
			},
			expectedResponseCode:    http.StatusOK,
			verifyCStorRestoreCount: true,
			isV1Version:             true,
		},
	}
	start := make(chan int, 1)
	defer close(start)
	go f.fakeCVCRoutine(start)
	// Started the channel
	<-start

	os.Setenv(util.OpenEBSNamespace, "openebs")
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			var replicaCount int
			f.fakeNodeCreator(5)
			f.fakePoolsCreator(test.cspcName, []string{test.poolVersion}, 5)
			if test.cvcObj != nil {
				_, err := f.openebsClient.CstorV1().CStorVolumeConfigs(namespace).Create(context.TODO(), test.cvcObj, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("Failed to create CVC %s", test.cvcObj.Name)
				}
				replicaCount = test.cvcObj.Spec.Provision.ReplicaCount
			}
			// Create storageclass
			if test.storageClass != nil {
				_, err := f.k8sClient.StorageV1().StorageClasses().Create(context.TODO(), test.storageClass, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("Failed to create SC %s", test.storageClass.Name)
				}
				replicaCount, _ = strconv.Atoi(test.storageClass.Parameters["replicaCount"])
			}
			// Create HTTPServer
			httpServer := &HTTPServer{
				cvcServer: NewCVCServer(server.DefaultServerConfig(), os.Stdout).
					WithOpenebsClientSet(f.openebsClient).
					WithKubernetesClientSet(f.k8sClient).
					WithSnapshotter(&snapshot.FakeSnapshotter{}),
				logger: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
			}

			//Marshal serializes the value provided into a json document
			jsonValue, _ := json.Marshal(test.cstorRestore)

			// Create a request to pass to handler
			req, _ := http.NewRequest("POST", "/latest/restore/", bytes.NewBuffer(jsonValue))
			// create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			rr := httptest.NewRecorder()
			req.Header.Add("Content-Type", "application/json")
			handler := http.HandlerFunc(httpServer.wrap(httpServer.restoreV1alpha1SpecificRequest))
			handler.ServeHTTP(rr, req)
			// Verify all the required results
			if rr.Code != test.expectedResponseCode {
				data, _ := ioutil.ReadAll(rr.Body)
				t.Errorf("failed to create restore for volume %s return code %d error: %s",
					test.cstorRestore.Spec.RestoreSrc, rr.Code, string(data))
			}
			if test.verifyCStorRestoreCount {
				var gotReplicaCount int
				restoreLabelSelector := metav1.ListOptions{
					LabelSelector: cstortypes.PersistentVolumeLabelKey + "=" + test.cstorRestore.Spec.VolumeName,
				}
				if test.isV1Version {
					restoreList, err := f.openebsClient.CstorV1().CStorRestores(namespace).List(context.TODO(), restoreLabelSelector)
					if err != nil {
						t.Errorf("Failed to list cstorRestore of volume %s", test.cvcObj.Name)
					}
					gotReplicaCount = len(restoreList.Items)
				} else {
					restoreList, err := f.openebsClient.OpenebsV1alpha1().CStorRestores(namespace).List(context.TODO(), restoreLabelSelector)
					if err != nil {
						t.Errorf("Failed to list cstorRestore of volume %s", test.cvcObj.Name)
					}
					gotReplicaCount = len(restoreList.Items)
				}

				if gotReplicaCount != replicaCount {
					t.Errorf(
						"Expected CStorRestore count %d but got %d",
						replicaCount, gotReplicaCount)
				}
			}
		})
	}
}

func TestRestoreWithDifferentPoolVersions(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	f.fakeNodeCreator(5)
	tests := map[string]struct {
		// cspcName used to create fake cstor pools
		cspcName     string
		poolVersions []string
		// cstorRestore is used to create restore
		cstorRestore              *cstorapis.CStorRestore
		cvcObj                    *cstorapis.CStorVolumeConfig
		storageClass              *storagev1.StorageClass
		v1CStorRestoreCount       int
		v1Alpha1CStorRestoreCount int
		expectedResponseCode      int
	}{
		//cspi.Status.Phase = cstorapis.CStorPoolStatusOnline
		"TestCase1": {
			cspcName:     "cspc-cstor-pool1",
			poolVersions: []string{"1.12.0", "2.0.3", "2.4.2"},
			cstorRestore: &cstorapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: cstorapis.CStorRestoreSpec{
					RestoreName: "restore1",
					VolumeName:  "volume1",
					RestoreSrc:  "127.0.0.1:3422",
				},
			},
			cvcObj: &cstorapis.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "volume1",
					Namespace: namespace,
					Labels: map[string]string{
						openebstypes.CStorPoolClusterLabelKey: "cspc-cstor-pool1",
					},
				},
				Spec: cstorapis.CStorVolumeConfigSpec{
					Provision: cstorapis.VolumeProvision{
						ReplicaCount: 3,
					},
				},
				Status: cstorapis.CStorVolumeConfigStatus{
					Phase: cstorapis.CStorVolumeConfigPhasePending,
				},
			},
			expectedResponseCode:      http.StatusOK,
			v1CStorRestoreCount:       1,
			v1Alpha1CStorRestoreCount: 2,
		},
		"TestCase2": {
			cspcName:     "cspc-cstor-pool2",
			poolVersions: []string{"1.12.0", "12.4.3", "2.4.2"},
			cstorRestore: &cstorapis.CStorRestore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: cstorapis.CStorRestoreSpec{
					RestoreName: "restore2",
					VolumeName:  "volume2",
					RestoreSrc:  "127.0.0.1:3422",
				},
			},
			cvcObj: &cstorapis.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "volume2",
					Namespace: namespace,
					Labels: map[string]string{
						openebstypes.CStorPoolClusterLabelKey: "cspc-cstor-pool2",
					},
				},
				Spec: cstorapis.CStorVolumeConfigSpec{
					Provision: cstorapis.VolumeProvision{
						ReplicaCount: 3,
					},
				},
				Status: cstorapis.CStorVolumeConfigStatus{
					Phase: cstorapis.CStorVolumeConfigPhasePending,
				},
			},
			expectedResponseCode:      http.StatusOK,
			v1CStorRestoreCount:       2,
			v1Alpha1CStorRestoreCount: 1,
		},
	}
	start := make(chan int, 1)
	defer close(start)
	go f.fakeCVCRoutine(start)
	// Started the channel
	<-start

	os.Setenv(util.OpenEBSNamespace, "openebs")
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			f.fakeNodeCreator(5)
			f.fakePoolsCreator(test.cspcName, test.poolVersions, len(test.poolVersions))
			if test.cvcObj != nil {
				_, err := f.openebsClient.CstorV1().CStorVolumeConfigs(namespace).Create(context.TODO(), test.cvcObj, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("Failed to create CVC %s", test.cvcObj.Name)
				}
			}
			// Create storageclass
			if test.storageClass != nil {
				_, err := f.k8sClient.StorageV1().StorageClasses().Create(context.TODO(), test.storageClass, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("Failed to create SC %s", test.storageClass.Name)
				}
			}
			// Create HTTPServer
			httpServer := &HTTPServer{
				cvcServer: NewCVCServer(server.DefaultServerConfig(), os.Stdout).
					WithOpenebsClientSet(f.openebsClient).
					WithKubernetesClientSet(f.k8sClient).
					WithSnapshotter(&snapshot.FakeSnapshotter{}),
				logger: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
			}

			//Marshal serializes the value provided into a json document
			jsonValue, _ := json.Marshal(test.cstorRestore)

			// Create a request to pass to handler
			req, _ := http.NewRequest("POST", "/latest/restore/", bytes.NewBuffer(jsonValue))
			// create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			rr := httptest.NewRecorder()
			req.Header.Add("Content-Type", "application/json")
			handler := http.HandlerFunc(httpServer.wrap(httpServer.restoreV1alpha1SpecificRequest))
			handler.ServeHTTP(rr, req)
			// Verify all the required results
			if rr.Code != test.expectedResponseCode {
				data, _ := ioutil.ReadAll(rr.Body)
				t.Errorf("failed to create restore for volume %s return code %d error: %s",
					test.cstorRestore.Spec.RestoreSrc, rr.Code, string(data))
			}
			restoreLabelSelector := metav1.ListOptions{
				LabelSelector: cstortypes.PersistentVolumeLabelKey + "=" + test.cstorRestore.Spec.VolumeName,
			}
			v1RestoreList, err := f.openebsClient.CstorV1().CStorRestores(namespace).List(context.TODO(), restoreLabelSelector)
			if err != nil {
				t.Errorf("Failed to list cstorRestore of volume %s", test.cvcObj.Name)
			}
			v1Alpha1RestoreList, err := f.openebsClient.OpenebsV1alpha1().CStorRestores(namespace).List(context.TODO(), restoreLabelSelector)
			if err != nil {
				t.Errorf("Failed to list cstorRestore of volume %s", test.cvcObj.Name)
			}
			if len(v1RestoreList.Items) != test.v1CStorRestoreCount {
				t.Errorf("Expected %d count of v1 restore resources but got only %d",
					test.v1CStorRestoreCount, len(v1RestoreList.Items))
			}
			if len(v1Alpha1RestoreList.Items) != test.v1Alpha1CStorRestoreCount {
				t.Errorf("Expected %d count of v1 restore resources but got only %d",
					test.v1Alpha1CStorRestoreCount, len(v1Alpha1RestoreList.Items))
			}
		})
	}
}
