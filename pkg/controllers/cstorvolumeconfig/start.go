/*
Copyright 2017 The OpenEBS Authors

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
	"flag"
	"os"
	"os/signal"
	"time"

	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	informers "github.com/openebs/api/v3/pkg/client/informers/externalversions"
	leader "github.com/openebs/api/v3/pkg/kubernetes/leaderelection"
	server "github.com/openebs/cstor-operators/pkg/server"
	cvcserver "github.com/openebs/cstor-operators/pkg/server/cstorvolumeconfig"
	"github.com/openebs/cstor-operators/pkg/snapshot"
	"github.com/pkg/errors"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var (
	// lease lock resource name for lease API resource
	leaderElectionLockName = "cvc-controller-leader"
	// port on which CVC server serve the REST request
	port = 5757
)

// Command line flags
var (
	kubeconfig              = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Required only when running out of cluster.")
	leaderElection          = flag.Bool("leader-election", false, "Enables leader election.")
	leaderElectionNamespace = flag.String("leader-election-namespace", "", "The namespace where the leader election resource exists. Defaults to the pod namespace if not set.")
	bindAddr                = flag.String("bind", "", "IP Address to bind for CVC-Operator Server")
)

// ServerOptions holds information to start the CVC server
type ServerOptions struct {
	// httpServer holds the CVC Server configurations
	httpServer *cvcserver.HTTPServer
}

// Start starts the cstorvolumeclaim controller.
func Start() error {

	klog.InitFlags(nil)
	err := flag.Set("logtostderr", "true")
	if err != nil {
		return errors.Wrap(err, "failed to set logtostderr flag")
	}
	flag.Parse()

	// Get in cluster config
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

	// setupCVCServer instantiate the HTTP server to serve the CVC request
	srvOptions, err := setupCVCServer(kubeClient, openebsClient)
	if err != nil {
		return errors.Wrapf(err, "failed to setupCVCServer")
	}
	defer srvOptions.httpServer.Shutdown()

	// openebsNamespace will hold where the OpenEBS is installed
	openebsNamespace = getNamespace()
	if openebsNamespace == "" {
		return errors.Errorf("failed to get openebs namespace got empty")
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	cvcInformerFactory := informers.NewSharedInformerFactory(openebsClient, time.Second*30)

	// Build() fn of all controllers calls AddToScheme to adds all types of this
	// clientset into the given scheme.
	// If multiple controllers happen to call this AddToScheme same time,
	// it causes panic with error saying concurrent map access.
	// This lock is used to serialize the AddToScheme call of all controllers.
	controller, err := NewCVCControllerBuilder().
		withKubeClient(kubeClient).
		withOpenEBSClient(openebsClient).
		//withNDMClient(ndmClient).
		withCVCSynced(cvcInformerFactory).
		withCVCLister(cvcInformerFactory).
		withCVLister(cvcInformerFactory).
		withCVRLister(cvcInformerFactory).
		withCVRInformerSync(cvcInformerFactory).
		withCVCStore().
		withRecorder(kubeClient).
		withEventHandler(cvcInformerFactory).
		withWorkqueueRateLimiting().Build()

	if err != nil {
		return errors.Wrapf(err, "error building controller instance")
	}

	// Threadiness defines the number of workers to be launched in Run function
	run := func(context.Context) {
		// run...
		stopCh := make(chan struct{})
		kubeInformerFactory.Start(stopCh)
		cvcInformerFactory.Start(stopCh)
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
	return rest.InClusterConfig()
}

// setupCVCServer will load the required server configuration and start the CVC server
func setupCVCServer(k8sclientset kubernetes.Interface, openebsClientset clientset.Interface) (*ServerOptions, error) {
	options := &ServerOptions{}
	// Load default server config
	config := server.DefaultServerConfig()

	// Update BindAddress if address is provided as a option
	if bindAddr != nil && *bindAddr != "" {
		config.BindAddr = *bindAddr
	} else {
		klog.Fatalln("bindAddr does not have an IP configure, check the `bind` container arg")
	}

	config.Port = &port
	err := config.NormalizeAddrs()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to setup CVC Server")
	}

	cvcServer := cvcserver.NewCVCServer(config, os.Stdout).
		WithOpenebsClientSet(openebsClientset).
		WithKubernetesClientSet(k8sclientset).
		WithSnapshotter(&snapshot.SnapClient{})

	// Setup the HTTP server
	http, err := cvcserver.NewHTTPServer(cvcServer)
	if err != nil {
		cvcServer.Shutdown()
		klog.Errorf("failed to start http server: %+v", err)
		return nil, err
	}
	options.httpServer = http
	return options, nil
}
