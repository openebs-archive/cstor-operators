package cstorvolumeconfig

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

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
				"app":                                "cstor-pool",
				cstortypes.CStorPoolInstanceLabelKey: cspi.Name,
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
			WithName(volumeName + "-" + cspiList.Items[0].Name).
			WithLabelsNew(labels).
			WithStatusPhase(phase)
		_, err := f.openebsClient.CstorV1().CStorVolumeReplicas(namespace).Create(cvr)
		if err != nil {
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
		WithName(volumeName).
		WithLabelsNew(labels)
	_, err := f.openebsClient.CstorV1().CStorVolumes(namespace).Create(cv)
	if err != nil {
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
		// snapshoter is used to mock snapshot operations on volumes
		snapshoter *snapshot.FakeSnapshoter
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
			snapshoter:                           &snapshot.FakeSnapshoter{},
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
			snapshoter: &snapshot.FakeSnapshoter{
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
			snapshoter: &snapshot.FakeSnapshoter{
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
			snapshoter:           &snapshot.FakeSnapshoter{},
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
			snapshoter:           &snapshot.FakeSnapshoter{},
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
			snapshoter:           &snapshot.FakeSnapshoter{},
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
			snapshoter: &snapshot.FakeSnapshoter{
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
					WithSnapshoter(test.snapshoter),
				logger: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
			}
			//Marshal serializes the value provided into a json document
			jsonValue, err := json.Marshal(test.cstorBackup)
			if err != nil {
				t.Errorf("failed to marshal cstor backup error: %s", err.Error())
			}
			// Create a request to pass to our handler
			req, err := http.NewRequest("POST", "/latest/volumes/", bytes.NewBuffer(jsonValue))
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
					WithSnapshoter(&snapshot.FakeSnapshoter{}),
				logger: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
			}
			//Marshal serializes the value provided into a json document
			jsonValue, _ := json.Marshal(test.cstorBackup)
			// Create a request to pass to handler
			req, _ := http.NewRequest("POST", "/latest/volumes/", bytes.NewBuffer(jsonValue))
			// create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			rr := httptest.NewRecorder()
			req.Header.Add("Content-Type", "application/json")
			handler := http.HandlerFunc(httpServer.wrap(httpServer.backupV1alpha1SpecificRequest))
			handler.ServeHTTP(rr, req)
			// Verify all the required results
			if rr.Code != http.StatusOK {
				data, _ := ioutil.ReadAll(rr.Body)
				t.Errorf("failed to create backup for volume %s return code %d error: %s",
					test.cstorBackup.Spec.VolumeName, rr.Code, string(data))
			}
			// ================================= Change according to the test configurations =========================
			backupObj, err := f.openebsClient.OpenebsV1alpha1().CStorBackups(namespace).Get(backupName, metav1.GetOptions{})
			if err == nil && test.shouldMarkPoolPodDown {
				cspiName := backupObj.Labels[cstortypes.CStorPoolInstanceNameLabelKey]
				poolManager, err := fetchPoolManagerFromCSPI(f.k8sClient, cspiName)
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
				cspiName := backupObj.Labels[cstortypes.CStorPoolInstanceNameLabelKey]
				poolManager, err := fetchPoolManagerFromCSPI(f.k8sClient, cspiName)
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
			rr = httptest.NewRecorder()
			req.Header.Add("Content-Type", "application/json")
			// Create a request to pass to handler
			req, err = http.NewRequest("GET", "/latest/volumes/", bytes.NewBuffer(jsonValue))
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

// TODO: Add more test code for DELETE method in backup end point and Restore end point
