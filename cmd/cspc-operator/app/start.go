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

package app

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"time"

	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	informers "github.com/openebs/api/v3/pkg/client/informers/externalversions"
	leader "github.com/openebs/api/v3/pkg/kubernetes/leaderelection"
	cspccontroller "github.com/openebs/cstor-operators/pkg/controllers/cspc-controller"
	"github.com/pkg/errors"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var (
	kubeconfig = flag.String("kubeconfig", "", "Path for kube config")
	// lease lock resource name for lease API resource
	leaderElectionLockName = "cspc-controller-leader"
)

// Command line flags
var (
	leaderElection          = flag.Bool("leader-election", false, "Enables leader election.")
	leaderElectionNamespace = flag.String("leader-election-namespace", "", "The namespace where the leader election resource exists. Defaults to the pod namespace if not set.")
)

const (
	// ResyncInterval is sync interval of the watcher
	ResyncInterval = 30 * time.Second
)

// Start starts the cstor-operator.
func Start() error {
	klog.InitFlags(nil)
	err := flag.Set("logtostderr", "true")
	if err != nil {
		return errors.Wrap(err, "failed to set logtostderr flag")
	}
	flag.Parse()

	cfg, err := getClusterConfig(*kubeconfig)
	if err != nil {
		return errors.Wrap(err, "error building kubeconfig")
	}

	// Building Kubernetes Clientset
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "error building kubernetes clientset")
	}

	// Building OpenEBS Clientset
	openebsClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "error building openebs clientset")
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, getSyncInterval())
	cspcInformerFactory := informers.NewSharedInformerFactory(openebsClient, getSyncInterval())
	// Build() fn of all controllers calls AddToScheme to adds all types of this
	// clientset into the given scheme.
	// If multiple controllers happen to call this AddToScheme same time,
	// it causes panic with error saying concurrent map access.
	// This lock is used to serialize the AddToScheme call of all controllers.
	//controllerMtx.Lock()

	controller, err := cspccontroller.NewControllerBuilder().
		WithKubeClient(kubeClient).
		WithOpenEBSClient(openebsClient).
		WithCSPCSynced(cspcInformerFactory).
		WithCSPCLister(cspcInformerFactory).
		WithRecorder(kubeClient).
		WithEventHandler(cspcInformerFactory).
		WithWorkqueueRateLimiting().Build()

	// blocking call, can't use defer to release the lock
	//controllerMtx.Unlock()

	if err != nil {
		return errors.Wrapf(err, "error building controller instance")
	}

	run := func(context.Context) {
		stopCh := make(chan struct{})
		kubeInformerFactory.Start(stopCh)
		cspcInformerFactory.Start(stopCh)
		go controller.Run(2, stopCh)

		// ...until SIGINT
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		close(stopCh)
	}

	if !*leaderElection {
		run(context.TODO())
	} else {
		le := leader.NewLeaderElection(kubeClient, leaderElectionLockName, run)
		if *leaderElectionNamespace != "" {
			le.WithNamespace(*leaderElectionNamespace)
		}
		if err := le.Run(); err != nil {
			klog.Fatalf("failed to initialize leader election: %v", err)
		}
	}
	return nil
}

// GetClusterConfig return the config for k8s.
func getClusterConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	klog.V(2).Info("Kubeconfig flag is empty")
	return rest.InClusterConfig()
}

// getSyncInterval gets the resync interval from environment variable.
// If missing or zero then default to 30 seconds
// otherwise return the obtained value
func getSyncInterval() time.Duration {
	resyncInterval, err := strconv.Atoi(os.Getenv("RESYNC_INTERVAL"))
	if err != nil || resyncInterval == 0 {
		klog.Warningf("Incorrect resync interval %q obtained from env, defaulting to %q seconds", resyncInterval, ResyncInterval)
		return ResyncInterval
	}
	return time.Duration(resyncInterval) * time.Second
}
