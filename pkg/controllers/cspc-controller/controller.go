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

package cspccontroller

import (
	"fmt"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	types "github.com/openebs/api/v3/pkg/apis/types"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	openebsScheme "github.com/openebs/api/v3/pkg/client/clientset/versioned/scheme"
	cstorstoredversion "github.com/openebs/api/v3/pkg/client/clientset/versioned/typed/cstor/v1"
	openebsstoredversion "github.com/openebs/api/v3/pkg/client/clientset/versioned/typed/openebs.io/v1alpha1"
	informers "github.com/openebs/api/v3/pkg/client/informers/externalversions"
	v1interface "github.com/openebs/api/v3/pkg/client/informers/externalversions/cstor/v1"
	listers "github.com/openebs/api/v3/pkg/client/listers/cstor/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "cspc-controller"

// Controller is the controller implementation for CSPC resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	// clientset is a openebs custom resource package generated for custom API group.
	clientset clientset.Interface

	// ndmclientset is a ndm custom resource package generated for custom API group.
	//ndmclientset ndmclientset.Interface

	// cspcLister can list/get cspc from the shared informer's store
	cspcLister listers.CStorPoolClusterLister

	// rsLister can list/get replica sets from the shared informer's  store
	cspiLister listers.CStorPoolInstanceLister

	// cspcSynced returns true if the cspc store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	cspcSynced cache.InformerSynced

	// cspiSynced returns true if the cspc store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	cspiSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// To allow injection of syncCSPC for testing.
	syncHandler func(cspcKey string) error

	// used for unit testing
	enqueueCSPC func(cspc *cstor.CStorPoolCluster)
}

// ControllerBuilder is the builder object for controller.
type ControllerBuilder struct {
	Controller *Controller
}

// NewControllerBuilder returns an empty instance of controller builder.
func NewControllerBuilder() *ControllerBuilder {
	return &ControllerBuilder{
		Controller: &Controller{},
	}
}

// withKubeClient fills kube client to controller object.
func (cb *ControllerBuilder) WithKubeClient(ks kubernetes.Interface) *ControllerBuilder {
	cb.Controller.kubeclientset = ks
	return cb
}

// withOpenEBSClient fills openebs client to controller object.
func (cb *ControllerBuilder) WithOpenEBSClient(cs clientset.Interface) *ControllerBuilder {
	cb.Controller.clientset = cs
	return cb
}

// withSpcLister fills cspc lister to controller object.
func (cb *ControllerBuilder) WithCSPCLister(sl informers.SharedInformerFactory) *ControllerBuilder {
	cspcInformer := GetVersionedCSPCInterface(sl).CStorPoolClusters()
	cb.Controller.cspcLister = cspcInformer.Lister()
	return cb
}

// WithCSPILister fills cspi lister to controller object.
func (cb *ControllerBuilder) WithCSPILister(sl informers.SharedInformerFactory) *ControllerBuilder {
	cspiInformer := GetStoredCSPIVersionInterface(sl).CStorPoolInstances()
	cb.Controller.cspiLister = cspiInformer.Lister()
	return cb
}

// withcspcSynced adds object sync information in cache to controller object.
func (cb *ControllerBuilder) WithCSPCSynced(sl informers.SharedInformerFactory) *ControllerBuilder {
	cspcInformer := GetVersionedCSPCInterface(sl).CStorPoolClusters()
	cb.Controller.cspcSynced = cspcInformer.Informer().HasSynced
	return cb
}

// WithCSPISynced adds object sync information in cache to controller object.
func (cb *ControllerBuilder) WithCSPISynced(sl informers.SharedInformerFactory) *ControllerBuilder {
	cspiInformer := GetStoredCSPIVersionInterface(sl).CStorPoolInstances()
	cb.Controller.cspcSynced = cspiInformer.Informer().HasSynced
	return cb
}

// withWorkqueue adds workqueue to controller object.
func (cb *ControllerBuilder) WithWorkqueueRateLimiting() *ControllerBuilder {
	cb.Controller.workqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "CSPC")
	return cb
}

// withRecorder adds recorder to controller object.
func (cb *ControllerBuilder) WithRecorder(ks kubernetes.Interface) *ControllerBuilder {
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	// eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: ks.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	cb.Controller.recorder = recorder
	return cb
}

// withEventHandler adds event handlers controller object.
func (cb *ControllerBuilder) WithEventHandler(InformerFactory informers.SharedInformerFactory) *ControllerBuilder {
	cspcInformer := GetVersionedCSPCInterface(InformerFactory).CStorPoolClusters()
	//Set up an event handler for when CSPC resources change
	cspcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cb.Controller.addCSPC,
		UpdateFunc: cb.Controller.updateCSPC,
		// This will enter the sync loop and no-op, because the cspc has been deleted from the store.
		DeleteFunc: cb.Controller.deleteCSPC,
	})
	return cb
}

func (cb *ControllerBuilder) withDefaults() {
	cb.Controller.syncHandler = cb.Controller.syncCSPC
	cb.Controller.enqueueCSPC = cb.Controller.enqueue
}

// Build returns a controller instance.
func (cb *ControllerBuilder) Build() (*Controller, error) {
	cb.withDefaults()
	err := openebsScheme.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}
	return cb.Controller, nil
}

// addCSPC is the add event handler for cspc
func (c *Controller) addCSPC(obj interface{}) {
	cspc, ok := obj.(*cstor.CStorPoolCluster)
	if !ok {
		runtime.HandleError(fmt.Errorf("Couldn't get cspc object %#v", obj))
		return
	}
	if cspc.Annotations[string(types.OpenEBSDisableReconcileLabelKey)] == "true" {
		message := fmt.Sprintf("reconcile is disabled via %q annotation", string(types.OpenEBSDisableReconcileLabelKey))
		c.recorder.Event(cspc, corev1.EventTypeWarning, "Create", message)
		return
	}
	klog.V(4).Infof("Queuing CSPC %s for add event", cspc.Name)
	c.enqueueCSPC(cspc)
}

// updateCSPC is the update event handler for cspc.
func (c *Controller) updateCSPC(oldCSPC, newCSPC interface{}) {
	cspc, ok := newCSPC.(*cstor.CStorPoolCluster)
	if !ok {
		runtime.HandleError(fmt.Errorf("Couldn't get cspc object %#v", newCSPC))
		return
	}
	if cspc.Annotations[string(types.OpenEBSDisableReconcileLabelKey)] == "true" {
		message := fmt.Sprintf("reconcile is disabled via %q annotation", string(types.OpenEBSDisableReconcileLabelKey))
		c.recorder.Event(cspc, corev1.EventTypeWarning, "Update", message)
		return
	}
	c.enqueueCSPC(cspc)
}

// deleteCSPC is the delete event handler for cspc.
func (c *Controller) deleteCSPC(obj interface{}) {
	cspc, ok := obj.(*cstor.CStorPoolCluster)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		cspc, ok = tombstone.Obj.(*cstor.CStorPoolCluster)
		if !ok {
			runtime.HandleError(fmt.Errorf("Tombstone contained object that is not a cstorpoolcluster %#v", obj))
			return
		}
	}
	if cspc.Annotations[string(types.OpenEBSDisableReconcileLabelKey)] == "true" {
		message := fmt.Sprintf("reconcile is disabled via %q annotation", string(types.OpenEBSDisableReconcileLabelKey))
		c.recorder.Event(cspc, corev1.EventTypeWarning, "Delete", message)
		return
	}
	klog.V(4).Infof("Deleting cstorpoolcluster %s", cspc.Name)
	c.enqueueCSPC(cspc)
}

func GetVersionedCSPCInterface(cspcInformerFactory informers.SharedInformerFactory) v1interface.Interface {
	return cspcInformerFactory.Cstor().V1()
}

func GetStoredCSPIVersionInterface(cspiInformerFactory informers.SharedInformerFactory) v1interface.Interface {
	return cspiInformerFactory.Cstor().V1()
}

func (c *Controller) GetStoredCStorVersionClient() cstorstoredversion.CstorV1Interface {
	return c.clientset.CstorV1()
}

func (c *Controller) GetStoredOpenebsVersionClient() openebsstoredversion.OpenebsV1alpha1Interface {
	return c.clientset.OpenebsV1alpha1()
}
