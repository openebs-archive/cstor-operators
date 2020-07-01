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

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	openebsapis "github.com/openebs/api/pkg/apis/openebs.io/v1alpha1"
	cstortypes "github.com/openebs/api/pkg/apis/types"
	"github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"

	openebstypes "github.com/openebs/api/pkg/apis/types"
	clientset "github.com/openebs/api/pkg/client/clientset/versioned"
	openebsFakeClientset "github.com/openebs/api/pkg/client/clientset/versioned/fake"
	"github.com/openebs/api/pkg/util"
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
		_, err := f.k8sClient.CoreV1().Nodes().Create(node)
		if err != nil && !k8serror.IsAlreadyExists(err) {
			klog.Error(err)
			continue
		}
		_, err = f.k8sClient.CoreV1().Nodes().Update(node)
		if err != nil {
			klog.Error(err)
		}
	}
}

func (f *fixture) fakePoolsCreator(cspcName string, poolCount int) error {
	nodeList, err := f.k8sClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(nodeList.Items) < poolCount {
		return errors.Errorf("enough nodes doesn't exist to create fake CSPIs")
	}
	for i := 0; i < poolCount; i++ {
		labels := map[string]string{
			openebstypes.HostNameLabelKey:         nodeList.Items[i].Name,
			openebstypes.CStorPoolClusterLabelKey: cspcName,
		}
		cspi := cstor.NewCStorPoolInstance().
			WithName(cspcName + "-" + rand.String(4)).
			WithNamespace(namespace).
			WithNodeSelectorByReference(nodeList.Items[i].Labels).
			WithNodeName(nodeList.Items[i].Name).
			WithLabelsNew(labels)
		cspi.Status.Phase = cstor.CStorPoolStatusOnline
		cspiObj, err := f.openebsClient.CstorV1().CStorPoolInstances(namespace).Create(cspi)
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
func (f *fixture) createFakePoolPod(cspi *cstor.CStorPoolInstance) error {
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
	_, err := f.k8sClient.CoreV1().Pods(namespace).Create(pod)
	if err != nil {
		return err
	}
	return nil
}

// createFakeVolumeReplicas will create fake CVRs on cstorPools
func (f *fixture) createFakeVolumeReplicas(
	cspcName, volumeName string, replicaCount int, phase cstor.CStorVolumeReplicaPhase) error {
	cspiList, err := f.openebsClient.CstorV1().CStorPoolInstances(namespace).List(metav1.ListOptions{
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
		cvr := cstor.NewCStorVolumeReplica().
			WithName(volumeName + "-" + cspiList.Items[i].Name).
			WithLabelsNew(labels).
			WithStatusPhase(phase)
		_, err := f.openebsClient.CstorV1().CStorVolumeReplicas(namespace).Create(cvr)
		if err != nil && !k8serror.IsNotFound(err) {
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
	cv := cstor.NewCStorVolume().
		WithNamespace(namespace).
		WithName(volumeName).
		WithLabelsNew(labels)
	_, err := f.openebsClient.CstorV1().CStorVolumes(namespace).Create(cv)
	if err != nil && !k8serror.IsNotFound(err) {
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
	_, err := f.openebsClient.OpenebsV1alpha1().CStorCompletedBackups(bk.Namespace).Create(bk)
	return err
}

func (f *fixture) fakeCVCRoutine(channel chan int) {
	fmt.Printf("Fake CVC routine has started")
	channel <- 1
	for {
		cvcList, err := f.openebsClient.CstorV1().CStorVolumeConfigs(namespace).List(metav1.ListOptions{})
		if err != nil {
			klog.Error(err)
		}

		for _, cvcObj := range cvcList.Items {
			if cvcObj.Annotations[openebstypes.OpenEBSDisableReconcileLabelKey] == "true" {
				klog.Infof("Skipping Reconcilation for CVC %s", cvcObj.Name)
				continue
			}
			if cvcObj.Status.Phase == cstor.CStorVolumeConfigPhasePending {
				cspcName := cvcObj.Labels[string(openebstypes.CStorPoolClusterLabelKey)]
				err := f.createFakeVolumeReplicas(cspcName, cvcObj.Name, cvcObj.Spec.Provision.ReplicaCount, cstor.CVRStatusOnline)
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
				cvcObj.Status.Phase = cstor.CStorVolumeConfigPhaseBound
				_, err = f.openebsClient.CstorV1().CStorVolumeConfigs(namespace).Update(&cvcObj)
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

func verifyExistenceOfPendingBackup(name, namespace string, openebsClient clientset.Interface) error {
	backUp, err := openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(name, metav1.GetOptions{})
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

func backupShouldNotExist(name, namespace string, openebsClient clientset.Interface) error {
	_, err := openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(name, metav1.GetOptions{})
	if k8serror.IsNotFound(err) {
		return nil
	}
	return err
}

func TestBackupPostEndPoint(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	f.fakeNodeCreator(5)
	tests := map[string]struct {
		// cspcName used to create fake cstor pools
		cspcName string
		// cstorBackup used to query on backup endpoint
		cstorBackup *openebsapis.CStorBackup
		// snapshotter is used to mock snapshot operations on volumes
		snapshotter *snapshot.FakeSnapshotter
		// cvrStatus creates CVR with provided phase
		cvrStatus                            cstor.CStorVolumeReplicaPhase
		isScheduledBackup                    bool
		expectedResponseCode                 int
		verifyBackUpStatus                   func(name, namespace string, openebsClient clientset.Interface) error
		checkExistenceOfCStorCompletedBackup bool
	}{
		"When all the resources exist and trigered backup endpoint with post method": {
			cspcName: "cspc-disk-pool1",
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
			cvrStatus:                            cstor.CVRStatusOnline,
			expectedResponseCode:                 http.StatusOK,
			verifyBackUpStatus:                   verifyExistenceOfPendingBackup,
			checkExistenceOfCStorCompletedBackup: true,
		},
		"When creation of snapshot fails": {
			cspcName: "cspc-disk-pool2",
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
			cvrStatus:            cstor.CVRStatusOnline,
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
			cvrStatus:            cstor.CVRStatusOnline,
		},
		"When cvrs are not healthy backup should fail": {
			cspcName: "cspc-disk-pool3",
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
			cvrStatus:            cstor.CVRStatusOffline,
		},
		"When request is to create scheduled backup": {
			cspcName: "cspc-disk-pool4",
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
			verifyBackUpStatus:   backupShouldNotExist,
			cvrStatus:            cstor.CVRStatusOnline,
			isScheduledBackup:    true,
		},
		"When local backup is requested": {
			cspcName: "cspc-disk-pool5",
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
			cvrStatus:            cstor.CVRStatusOffline,
		},
		"When local backup is requested and if there is failure in snapcreation": {
			cspcName: "cspc-disk-pool6",
			cstorBackup: &openebsapis.CStorBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: openebsapis.CStorBackupSpec{
					BackupName: "backup-local2",
					VolumeName: "volume-local2",
					SnapName:   "snapshot-local2",
					LocalSnap:  true,
				},
			},
			snapshotter: &snapshot.FakeSnapshotter{
				ShouldReturnFakeError: true,
			},
			expectedResponseCode: http.StatusBadRequest,
			verifyBackUpStatus:   backupShouldNotExist,
			cvrStatus:            cstor.CVRStatusOffline,
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
			f.fakePoolsCreator(test.cspcName, 5)
			f.createFakeVolumeReplicas(test.cspcName, test.cstorBackup.Spec.VolumeName, 3, test.cvrStatus)
			f.createFakeCStorVolume(test.cstorBackup.Spec.VolumeName)
			// If schedule backup is set then create CStorCompletedBackup(on assumption it is other than first request)
			if test.isScheduledBackup {
				f.createCStorCompletedBackup(test.cstorBackup, prevSnapshot)
			}
			// Create HTTPServer
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
				_, err := f.openebsClient.OpenebsV1alpha1().CStorCompletedBackups(namespace).Get(lastbkpName, metav1.GetOptions{})
				if err != nil {
					t.Errorf("failed to verify cstorcompleted backup existence error: %s", err.Error())
				}
			}
			if test.isScheduledBackup {
				backupObj, err := f.openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(backupName, metav1.GetOptions{})
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
			f.fakePoolsCreator(test.cspcName, 5)
			f.createFakeVolumeReplicas(test.cspcName, test.cstorBackup.Spec.VolumeName, 3, cstor.CVRStatusOnline)
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
			backupObj, err := f.openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(backupName, metav1.GetOptions{})
			if err == nil && test.shouldMarkPoolPodDown {
				cspiName := backupObj.Labels[openebstypes.CStorPoolInstanceNameLabelKey]
				poolManager, err := fetchPoolManagerFromCSPI(f.k8sClient, cspiName, namespace)
				if err == nil {
					for i, containerstatus := range poolManager.Status.ContainerStatuses {
						if containerstatus.Name == "cstor-pool-mgmt" {
							poolManager.Status.ContainerStatuses[i].Ready = false
						}
						f.k8sClient.CoreV1().Pods(namespace).Update(poolManager)
					}

				}
			}
			if err == nil && test.shouldMarkPoolPodNodeDown {
				cspiName := backupObj.Labels[openebstypes.CStorPoolInstanceNameLabelKey]
				poolManager, err := fetchPoolManagerFromCSPI(f.k8sClient, cspiName, namespace)
				if err == nil {
					node, err := f.k8sClient.CoreV1().Nodes().Get(poolManager.Spec.NodeName, metav1.GetOptions{})
					if err == nil {
						node.Status.Conditions = append(node.Status.Conditions, corev1.NodeCondition{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionFalse,
						})
						f.k8sClient.CoreV1().Nodes().Update(node)
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

			backupObj, err = f.openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(backupName, metav1.GetOptions{})
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
		cspcName string
		// cstorBackup used to query on backup endpoint
		cstorBackup                 *openebsapis.CStorBackup
		cstorCompletedBackup        *openebsapis.CStorCompletedBackup
		shouldBackupExists          bool
		shouldCompletedBackupExists bool
		shouldInjectSnapshotError   bool
		shouldInjectPayloadError    bool
		expectedResponseCode        int
	}{
		"When delete method is triggered on backup endpoint without any error injection": {
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
			shouldBackupExists:          false,
			shouldCompletedBackupExists: false,
			expectedResponseCode:        http.StatusOK,
		},
		"When delete method is triggered on backup endpoint by injecting error": {
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
			shouldBackupExists:          true,
			shouldCompletedBackupExists: false,
			shouldInjectSnapshotError:   true,
			expectedResponseCode:        http.StatusInternalServerError,
		},
		"Should not delete cstorcompleted backup": {
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
			cspcName: "cspc-cstor-pool4",
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
	}
	os.Setenv(util.OpenEBSNamespace, "openebs")
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			backupName := test.cstorBackup.Spec.SnapName + "-" + test.cstorBackup.Spec.VolumeName
			lastCompletedBackup := test.cstorBackup.Spec.BackupName + "-" + test.cstorBackup.Spec.VolumeName
			f.fakeNodeCreator(5)
			f.fakePoolsCreator(test.cspcName, 5)
			f.createFakeVolumeReplicas(test.cspcName, test.cstorBackup.Spec.VolumeName, 3, cstor.CVRStatusOnline)
			f.createFakeCStorVolume(test.cstorBackup.Spec.VolumeName)
			if test.cstorCompletedBackup != nil {
				_, err := f.openebsClient.OpenebsV1alpha1().CStorCompletedBackups(namespace).Create(test.cstorCompletedBackup)
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

			// Verifying CStorBackup
			backupObj, err := f.openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(backupName, metav1.GetOptions{})
			if err != nil {
				if !k8serror.IsNotFound(err) {
					t.Errorf("failed to get backup of volume %s error %v", test.cstorBackup.Spec.VolumeName, err)
				}
				if test.shouldBackupExists {
					t.Errorf("expected backup to exist but not found")
				}
			}
			if !test.shouldBackupExists && err == nil {
				t.Errorf("expected backup not to exist but found %s", backupObj.Name)
			}

			// Verifying CStorCompletedBackup
			completedBackupObj, err := f.openebsClient.OpenebsV1alpha1().CStorCompletedBackups(namespace).Get(lastCompletedBackup, metav1.GetOptions{})
			if err != nil {
				if !k8serror.IsNotFound(err) {
					t.Errorf("failed to get completed backup for volume %s error %v", test.cstorBackup.Spec.VolumeName, err)
				}
				if test.shouldCompletedBackupExists {
					t.Errorf("expected completedbackup to exist but not found")
				}
			}
			if !test.shouldCompletedBackupExists && err == nil {
				t.Errorf("expected completed backup not to exist but found %s", completedBackupObj.Name)
			}

		})
	}

}

func TestRestoreEndPoint(t *testing.T) {
	f := newFixture(t)
	f.SetFakeClient()
	tests := map[string]struct {
		// cspcName used to create fake cstor pools
		cspcName string
		// cstorRestore is used to create restore
		cstorRestore            *openebsapis.CStorRestore
		cvcObj                  *cstor.CStorVolumeConfig
		storageClass            *storagev1.StorageClass
		verifyCStorRestoreCount bool
		expectedResponseCode    int
	}{
		"When restore is triggered": {
			cspcName: "cspc-cstor-pool1",
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
			cvcObj: &cstor.CStorVolumeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "volume1",
					Namespace: namespace,
					Labels: map[string]string{
						openebstypes.CStorPoolClusterLabelKey: "cspc-cstor-pool1",
					},
				},
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					Phase: cstor.CStorVolumeConfigPhasePending,
				},
			},
			expectedResponseCode:    http.StatusOK,
			verifyCStorRestoreCount: true,
		},
		"When restore is triggered but CVC is not marked as bound": {
			cspcName: "cspc-cstor-pool2",
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
			cvcObj: &cstor.CStorVolumeConfig{
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
				Spec: cstor.CStorVolumeConfigSpec{
					Provision: cstor.VolumeProvision{
						ReplicaCount: 3,
					},
				},
				Status: cstor.CStorVolumeConfigStatus{
					Phase: cstor.CStorVolumeConfigPhasePending,
				},
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		"When restore source name is empty": {
			cspcName: "cspc-cstor-pool3",
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
			cspcName: "cspc-cstor-pool4",
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
			cspcName: "cspc-cstor-pool5",
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
			cspcName: "cspc-cstor-pool6",
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
			f.fakePoolsCreator(test.cspcName, 5)
			if test.cvcObj != nil {
				_, err := f.openebsClient.CstorV1().CStorVolumeConfigs(namespace).Create(test.cvcObj)
				if err != nil {
					t.Errorf("Failed to create CVC %s", test.cvcObj.Name)
				}
				replicaCount = test.cvcObj.Spec.Provision.ReplicaCount
			}
			// Create storageclass
			if test.storageClass != nil {
				_, err := f.k8sClient.StorageV1().StorageClasses().Create(test.storageClass)
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
				restoreList, err := f.openebsClient.
					OpenebsV1alpha1().
					CStorRestores(namespace).
					List(metav1.ListOptions{
						LabelSelector: cstortypes.PersistentVolumeLabelKey + "=" + test.cstorRestore.Spec.VolumeName})
				if err != nil {
					t.Errorf("Failed to list cstorRestore of volume %s", test.cvcObj.Name)
				}
				if len(restoreList.Items) != replicaCount {
					t.Errorf(
						"Expected CStorRestore count %d but got %d",
						replicaCount,
						len(restoreList.Items))
				}
			}
		})
	}
}
