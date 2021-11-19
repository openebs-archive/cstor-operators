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

package volumemgmt

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/openebs/cstor-operators/pkg/controllers/volume-mgmt/volume"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	informers "github.com/openebs/api/v3/pkg/client/informers/externalversions"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/openebs/api/v3/pkg/util/signals"
)

const (
	// NumThreads defines number of worker threads for resource watcher.
	NumThreads = 1
	// NumRoutinesThatFollow is for handling golang waitgroups.
	NumRoutinesThatFollow = 1
)

// DefaultSharedInformerInterval is used to sync watcher controller.
const DefaultSharedInformerInterval = 30 * time.Second

const (
	// CStorVolume is the controller to be watched
	CStorVolume = "cStorVolume"
	// EventMsgFormatter is the format string for event message generation
	EventMsgFormatter = "Volume is in %s state"
)

//EventReason is used as part of the Event reason when a resource goes through different phases
type EventReason string

const (
	// SuccessSynced is used as part of the Event 'reason' when a resource is synced
	SuccessSynced EventReason = "Synced"
	// MessageCreateSynced holds message for corresponding create request sync.
	MessageCreateSynced EventReason = "Received Resource create event"
	// MessageModifySynced holds message for corresponding modify request sync.
	MessageModifySynced EventReason = "Received Resource modify event"
	// MessageDestroySynced holds message for corresponding destroy request sync.
	MessageDestroySynced EventReason = "Received Resource destroy event"

	// SuccessCreated holds status for corresponding created resource.
	SuccessCreated EventReason = "Created"
	// MessageResourceCreated holds message for corresponding created resource.
	MessageResourceCreated EventReason = "Resource created successfully"

	// FailureCreate holds status for corresponding failed create resource.
	FailureCreate EventReason = "FailCreate"
	// MessageResourceFailCreate holds message for corresponding failed create resource.
	MessageResourceFailCreate EventReason = "Resource creation failed"

	// FailureUpdate holds status for corresponding failed update resource.
	FailureUpdate EventReason = "FailUpdate"

	// SuccessImported holds status for corresponding imported resource.
	SuccessImported EventReason = "Imported"
	// MessageResourceImported holds message for corresponding imported resource.
	MessageResourceImported EventReason = "Resource imported successfully"

	// FailureImport holds status for corresponding failed import resource.
	FailureImport EventReason = "FailImport"
	// MessageResourceFailImport holds message for corresponding failed import resource.
	MessageResourceFailImport EventReason = "Resource import failed"

	// FailureDestroy holds status for corresponding failed destroy resource.
	FailureDestroy EventReason = "FailDestroy"
	// MessageResourceFailDestroy holds message for corresponding failed destroy resource.
	MessageResourceFailDestroy EventReason = "Resource Destroy failed"

	// FailureValidate holds status for corresponding failed validate resource.
	FailureValidate EventReason = "FailValidate"
	// MessageResourceFailValidate holds message for corresponding failed validate resource.
	MessageResourceFailValidate EventReason = "Resource validation failed"

	// AlreadyPresent holds status for corresponding already present resource.
	AlreadyPresent EventReason = "AlreadyPresent"
	// MessageResourceAlreadyPresent holds message for corresponding already present resource.
	MessageResourceAlreadyPresent EventReason = "Resource already present"

	// SuccessUpdated holds status for corresponding updated resource.
	SuccessUpdated EventReason = "Updated"
	// MessageResourceUpdated holds message for corresponding updated resource.
	MessageResourceUpdated EventReason = "Resource updated successfully"
)

const (
	// CRDRetryInterval is used if CRD is not present.
	CRDRetryInterval = 10 * time.Second
	// ResourceWorkerInterval is used for resource sync.
	ResourceWorkerInterval = time.Second
)

//CStorVolumeStatus represents the status of a CStorVolume object
type CStorVolumeStatus string

// Status written onto CStorVolume objects.
const (
	// volume is getting initialized
	CVStatusInit CStorVolumeStatus = "Init"
	// volume allows IOs and snapshot
	CVStatusHealthy CStorVolumeStatus = "Healthy"
	// volume only satisfies consistency factor
	CVStatusDegraded CStorVolumeStatus = "Degraded"
	// Volume is offline
	CVStatusOffline CStorVolumeStatus = "Offline"
	// Error in retrieving volume details
	CVStatusError CStorVolumeStatus = "Error"
	// volume controller config generation failed due to invalid parameters
	CVStatusInvalid CStorVolumeStatus = "Invalid"
	// CR event ignored
	CVStatusIgnore CStorVolumeStatus = "Ignore"
)

// QueueLoad is for storing the key and type of operation before entering workqueue
type QueueLoad struct {
	Key       string // Key is the name of cstor volume given in metadata name field in the yaml
	Operation QueueOperation
}

// Environment is for environment variables passed for cstor-volume-mgmt.
type Environment string

const (
	// OpenEBSIOCStorVolumeID is the environment variable specified in pod.
	OpenEBSIOCStorVolumeID Environment = "OPENEBS_IO_CSTOR_VOLUME_ID"
)

//QueueOperation represents the type of operation on resource
type QueueOperation string

//Different type of operations on the controller
const (
	QOpAdd          QueueOperation = "add"
	QOpDestroy      QueueOperation = "destroy"
	QOpModify       QueueOperation = "modify"
	QOpPeriodicSync QueueOperation = "sync"
)

// namespace defines kubernetes namespace specified for cvr.
type namespace string

// Different types of k8s namespaces.
const (
	DefaultNameSpace namespace = "openebs"
)

// CheckForCStorVolumeCRD is Blocking call for checking status of CStorVolume CRD.
func CheckForCStorVolumeCRD(clientset clientset.Interface) {
	for {
		// Since this blocking function is restricted to check if CVR CRD is present
		// or not, we are trying to handle only the error of CVR CR List api indirectly.
		// CRD has only two types of scope, cluster and namespaced. If CR list api
		// for default namespace works fine, then CR list api works for all namespaces.
		_, err := clientset.CstorV1().CStorVolumes(string(DefaultNameSpace)).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			klog.Errorf("CStorVolume CRD not found. Retrying after %v, err : %v", CRDRetryInterval, err)
			time.Sleep(CRDRetryInterval)
			continue
		}
		klog.Info("CStorVolume CRD found")
		break
	}
}

// StartControllers instantiates CStorVolume controllers
// and watches them.
func StartControllers(kubeconfig string) {
	// Set up signals to handle the first shutdown signal gracefully.
	stopCh := signals.SetupSignalHandler()

	cfg, err := getClusterConfig(kubeconfig)
	if err != nil {
		klog.Fatalf(err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	openebsClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building openebs clientset: %s", err.Error())
	}

	volume.FileOperatorVar = util.RealFileOperator{}

	volume.UnixSockVar = util.RealUnixSock{}

	// Blocking call for checking status of istgt running in cstor-volume container.
	util.CheckForIscsi(volume.UnixSockVar)

	// Blocking call for checking status of CStorVolume CR.
	CheckForCStorVolumeCRD(openebsClient)

	// NewInformer returns a cache.Store and a controller for populating the store
	// while also providing event notifications. It’s basically a controller with some
	// boilerplate code to sync events from the FIFO queue to the downstream store.
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, getSyncInterval())
	openebsInformerFactory := informers.NewSharedInformerFactory(openebsClient, getSyncInterval())

	cStorVolumeController := NewCStorVolumeController(kubeClient, openebsClient, kubeInformerFactory,
		openebsInformerFactory)

	go kubeInformerFactory.Start(stopCh)
	go openebsInformerFactory.Start(stopCh)

	// Waitgroup for starting volume controller goroutines.
	var wg sync.WaitGroup
	wg.Add(NumRoutinesThatFollow)

	// Run controller for cStorVolume.
	go func() {
		if err = cStorVolumeController.Run(NumThreads, stopCh); err != nil {
			klog.Fatalf("Error running CStorVolume controller: %s", err.Error())
		}
		wg.Done()
	}()
	wg.Wait()
}

// GetClusterConfig return the config for k8s.
func getClusterConfig(kubeconfig string) (*rest.Config, error) {
	var masterURL string
	cfg, err := rest.InClusterConfig()
	if err != nil {
		klog.Errorf("Failed to get k8s Incluster config. %+v", err)
		if len(kubeconfig) == 0 {
			return nil, fmt.Errorf("kubeconfig is empty: %v", err.Error())
		}
		cfg, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("Error building kubeconfig: %s", err.Error())
		}
	}
	return cfg, err
}

// getSyncInterval gets the resync interval from environment variable.
// If missing or zero then default to DefaultSharedInformerInterval
// otherwise return the obtained value
func getSyncInterval() time.Duration {
	resyncInterval, err := strconv.Atoi(os.Getenv("RESYNC_INTERVAL"))
	if err != nil || resyncInterval == 0 {
		klog.Warningf("Incorrect resync interval %q obtained from env, defaulting to %q seconds", resyncInterval, DefaultSharedInformerInterval)
		return DefaultSharedInformerInterval
	}
	return time.Duration(resyncInterval) * time.Second
}
